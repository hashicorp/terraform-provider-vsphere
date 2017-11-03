package virtualdevice

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/dvportgroup"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/network"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/nsx"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

const (
	networkInterfaceSubresourceTypeE1000   = "e1000"
	networkInterfaceSubresourceTypeVmxnet3 = "vmxnet3"
	networkInterfaceSubresourceTypeUnknown = "unknown"
)

var networkInterfaceSubresourceTypeAllowedValues = []string{
	networkInterfaceSubresourceTypeE1000,
	networkInterfaceSubresourceTypeVmxnet3,
}

var networkInterfaceSubresourceMACAddressTypeAllowedValues = []string{
	string(types.VirtualEthernetCardMacTypeManual),
}

// NetworkInterfaceSubresourceSchema returns the schema for the disk
// sub-resource.
func NetworkInterfaceSubresourceSchema() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		// VirtualEthernetCardResourceAllocation
		"bandwidth_limit": {
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      -1,
			Description:  "The upper bandwidth limit of this network interface, in Mbits/sec.",
			ValidateFunc: validation.IntAtLeast(-1),
		},
		"bandwidth_reservation": {
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      0,
			Description:  "The bandwidth reservation of this network interface, in Mbits/sec.",
			ValidateFunc: validation.IntAtLeast(0),
		},
		"bandwidth_share_level": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      string(types.SharesLevelNormal),
			Description:  "The bandwidth share allocation level for this interface. Can be one of low, normal, high, or custom.",
			ValidateFunc: validation.StringInSlice(sharesLevelAllowedValues, false),
		},
		"bandwidth_share_count": {
			Type:         schema.TypeInt,
			Optional:     true,
			Computed:     true,
			Description:  "The share count for this network interface when the share level is custom.",
			ValidateFunc: validation.IntAtLeast(0),
		},

		// VirtualEthernetCard and friends
		"network_id": {
			Type:         schema.TypeString,
			Required:     true,
			Description:  "The ID of the network to connect this network interface to.",
			ValidateFunc: validation.NoZeroValues,
		},
		"adapter_type": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      networkInterfaceSubresourceTypeE1000,
			Description:  "The controller type. Can be one of e1000 or vmxnet3.",
			ValidateFunc: validation.StringInSlice(networkInterfaceSubresourceTypeAllowedValues, false),
		},
		"use_static_mac": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "If true, the mac_address field is treated as a static MAC address and set accordingly.",
		},
		"mac_address": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "The MAC address of this network interface. Can be manually set if use_static_mac is true.",
		},
	}
	structure.MergeSchema(s, subresourceSchema())
	return s
}

// NetworkInterfaceSubresource represents a vsphere_virtual_machine
// network_interface sub-resource, with a complex device lifecycle.
type NetworkInterfaceSubresource struct {
	*Subresource
}

// NewNetworkInterfaceSubresource returns a network_interface subresource
// populated with all of the necessary fields.
func NewNetworkInterfaceSubresource(client *govmomi.Client, rd *schema.ResourceData, d, old map[string]interface{}, idx int) *NetworkInterfaceSubresource {
	sr := &NetworkInterfaceSubresource{
		Subresource: &Subresource{
			schema:       NetworkInterfaceSubresourceSchema(),
			client:       client,
			srtype:       subresourceTypeNetworkInterface,
			data:         d,
			olddata:      old,
			resourceData: rd,
		},
	}
	sr.Index = idx
	return sr
}

// NetworkInterfaceApplyOperation processes an apply operation for all
// network_interfaces in the resource.
//
// The function takes the root resource's ResourceData, the provider
// connection, and the device list as known to vSphere at the start of this
// operation. All network_interface operations are carried out, with both the
// complete, updated, VirtualDeviceList, and the complete list of changes
// returned as a slice of BaseVirtualDeviceConfigSpec.
func NetworkInterfaceApplyOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) (object.VirtualDeviceList, []types.BaseVirtualDeviceConfigSpec, error) {
	o, n := d.GetChange(subresourceTypeNetworkInterface)
	ods := o.([]interface{})
	nds := n.([]interface{})

	var spec []types.BaseVirtualDeviceConfigSpec

	// Our old and new sets now have an accurate description of devices that may
	// have been added, removed, or changed. Look for removed devices first.
nextOld:
	for n, oe := range ods {
		om := oe.(map[string]interface{})
		for _, ne := range nds {
			nm := ne.(map[string]interface{})
			if om["key"] == nm["key"] {
				continue nextOld
			}
		}
		r := NewNetworkInterfaceSubresource(c, d, om, nil, n)
		dspec, err := r.Delete(l)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		l = applyDeviceChange(l, dspec)
		spec = append(spec, dspec...)
	}

	// Now check for creates and updates. The results of this operation are
	// committed to state after the operation completes.
	var updates []interface{}
nextNew:
	for n, ne := range nds {
		nm := ne.(map[string]interface{})
		for _, oe := range ods {
			om := oe.(map[string]interface{})
			if nm["key"] == om["key"] {
				// This is an update
				if reflect.DeepEqual(nm, om) {
					// no change is a no-op
					continue nextNew
				}
				r := NewNetworkInterfaceSubresource(c, d, nm, om, n)
				uspec, err := r.Update(l)
				if err != nil {
					return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
				}
				l = applyDeviceChange(l, uspec)
				spec = append(spec, uspec...)
				updates = append(updates, r.Data())
				continue nextNew
			}
		}
		r := NewNetworkInterfaceSubresource(c, d, nm, nil, n)
		cspec, err := r.Create(l)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		l = applyDeviceChange(l, cspec)
		spec = append(spec, cspec...)
		updates = append(updates, r.Data())
	}

	// We are now done! Return the updated device list and config spec. Save updates as well.
	if err := d.Set(subresourceTypeNetworkInterface, updates); err != nil {
		return nil, nil, err
	}
	return l, spec, nil
}

// NetworkInterfaceRefreshOperation processes a refresh operation for all of
// the disks in the resource.
//
// This functions similar to NetworkInterfaceApplyOperation, but nothing to
// change is returned, all necessary values are just set and committed to
// state.
func NetworkInterfaceRefreshOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) error {
	devices := l.Select(func(device types.BaseVirtualDevice) bool {
		if _, ok := device.(types.BaseVirtualEthernetCard); ok {
			return true
		}
		return false
	})
	curSet := d.Get(subresourceTypeNetworkInterface).([]interface{})
	var newSet []interface{}
	// First check for negative keys. These are freshly added devices that are
	// usually coming into read post-create.
	//
	// If we find what we are looking for, we remove the device from the working
	// set so that we don't try and process it in the next few passes.
	for n, item := range curSet {
		m := item.(map[string]interface{})
		if m["key"].(int) < 1 {
			r := NewNetworkInterfaceSubresource(c, d, m, nil, n)
			if err := r.Read(l); err != nil {
				return fmt.Errorf("%s: %s", r.Addr(), err)
			}
			newM := r.Data()
			if newM["key"].(int) < 1 {
				// This should not have happened - if it did, our device
				// creation/update logic failed somehow that we were not able to track.
				return fmt.Errorf("device %v with address %v still unaccounted for after update/read", newM["key"], newM["device_address"])
			}
			newSet = append(newSet, r.Data())
			for i := 0; i < len(devices); i++ {
				device := devices[i]
				if device.GetVirtualDevice().Key == int32(newM["key"].(int)) {
					devices = append(devices[:i], devices[i+1:]...)
				}
			}
		}
	}

	// Go over the remaining devices, refresh via key, and then remove their
	// entries as well.
	for i := 0; i < len(devices); i++ {
		device := devices[i]
		for n, item := range curSet {
			m := item.(map[string]interface{})
			if m["key"].(int) < 0 {
				// Skip any of these keys as we won't be matching any of those anyway here
				continue
			}
			if device.GetVirtualDevice().Key != int32(m["key"].(int)) {
				// Skip any device that doesn't match key as well
				continue
			}
			// We should have our device -> resource match, so read now.
			r := NewNetworkInterfaceSubresource(c, d, m, nil, n)
			if err := r.Read(l); err != nil {
				return fmt.Errorf("%s: %s", r.Addr(), err)
			}
			// Done reading, push this onto our new set and remove the device from
			// the list
			newSet = append(newSet, r.Data())
			devices = append(devices[:i], devices[i+1:]...)
		}
	}

	// Finally, any device that is still here is orphaned. They should be added
	// as new devices.
	for n, device := range devices {
		m := make(map[string]interface{})
		vd := device.GetVirtualDevice()
		ctlr := l.FindByKey(vd.ControllerKey)
		if ctlr == nil {
			return fmt.Errorf("could not find controller with key %d", vd.Key)
		}
		m["key"] = int(vd.Key)
		m["device_address"] = computeDevAddr(vd, ctlr.(types.BaseVirtualController))
		r := NewNetworkInterfaceSubresource(c, d, m, nil, n)
		if err := r.Read(l); err != nil {
			return fmt.Errorf("%s: %s", r.Addr(), err)
		}
		newSet = append(newSet, r.Data())
	}

	return d.Set(subresourceTypeNetworkInterface, newSet)
}

// baseVirtualEthernetCardToBaseVirtualDevice converts a
// BaseVirtualEthernetCard value into a BaseVirtualDevice.
func baseVirtualEthernetCardToBaseVirtualDevice(v types.BaseVirtualEthernetCard) types.BaseVirtualDevice {
	switch t := v.(type) {
	case *types.VirtualE1000:
		return types.BaseVirtualDevice(t)
	case *types.VirtualE1000e:
		return types.BaseVirtualDevice(t)
	case *types.VirtualPCNet32:
		return types.BaseVirtualDevice(t)
	case *types.VirtualSriovEthernetCard:
		return types.BaseVirtualDevice(t)
	case *types.VirtualVmxnet2:
		return types.BaseVirtualDevice(t)
	case *types.VirtualVmxnet3:
		return types.BaseVirtualDevice(t)
	}
	panic(fmt.Errorf("unknown ethernet card type %T", v))
}

// baseVirtualDeviceToBaseVirtualEthernetCard converts a BaseVirtualDevice
// value into a BaseVirtualEthernetCard.
func baseVirtualDeviceToBaseVirtualEthernetCard(v types.BaseVirtualDevice) (types.BaseVirtualEthernetCard, error) {
	switch t := v.(type) {
	case *types.VirtualE1000:
		return types.BaseVirtualEthernetCard(t), nil
	case *types.VirtualE1000e:
		return types.BaseVirtualEthernetCard(t), nil
	case *types.VirtualPCNet32:
		return types.BaseVirtualEthernetCard(t), nil
	case *types.VirtualSriovEthernetCard:
		return types.BaseVirtualEthernetCard(t), nil
	case *types.VirtualVmxnet2:
		return types.BaseVirtualEthernetCard(t), nil
	case *types.VirtualVmxnet3:
		return types.BaseVirtualEthernetCard(t), nil
	}
	return nil, fmt.Errorf("device is not a network device (%T)", v)
}

// Create creates a vsphere_virtual_machine network_interface sub-resource.
func (r *NetworkInterfaceSubresource) Create(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	var spec []types.BaseVirtualDeviceConfigSpec
	ctlr, err := r.ControllerForCreateUpdate(l, SubresourceControllerTypePCI, 0)
	if err != nil {
		return nil, err
	}

	// govmomi has helpers that allow the easy fetching of a network's backing
	// info, once we actually know what that backing is. Set all of that stuff up
	// now.
	net, err := network.FromID(r.client, r.Get("network_id").(string))
	if err != nil {
		return nil, err
	}
	bctx, bcancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer bcancel()
	backing, err := net.EthernetCardBackingInfo(bctx)
	if err != nil {
		return nil, err
	}
	device, err := l.CreateEthernetCard(r.Get("adapter_type").(string), backing)
	if err != nil {
		return nil, err
	}

	// CreateEthernetCard does not attach stuff, however, assuming that you will
	// let vSphere take care of the attachment and what not, as there is usually
	// only one PCI device per virtual machine and their tools don't really care
	// about state. Terraform does though, so we need to not only set but also
	// track that stuff.
	if err := r.assignEthernetCard(l, device, ctlr); err != nil {
		return nil, err
	}
	// Ensure the device starts connected
	l.Connect(device)

	// Set base-level card bits now
	card := device.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard()
	card.Key = l.NewKey()

	// Set the rest of the settings here.
	if r.Get("use_static_mac").(bool) {
		card.AddressType = string(types.VirtualEthernetCardMacTypeManual)
		card.MacAddress = r.Get("mac_address").(string)
	}
	alloc := &types.VirtualEthernetCardResourceAllocation{
		Limit:       structure.Int64Ptr(int64(r.Get("bandwidth_limit").(int))),
		Reservation: structure.Int64Ptr(int64(r.Get("bandwidth_reservation").(int))),
		Share: types.SharesInfo{
			Shares: int32(r.Get("bandwidth_share_count").(int)),
			Level:  types.SharesLevel(r.Get("bandwidth_share_level").(string)),
		},
	}
	card.ResourceAllocation = alloc

	// Done here. Save ID, push the device to the new device list and return.
	r.SaveDevIDs(device, ctlr)
	dspec, err := object.VirtualDeviceList{device}.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
	if err != nil {
		return nil, err
	}
	spec = append(spec, dspec...)
	return spec, nil
}

// Read reads a vsphere_virtual_machine network_interface sub-resource.
func (r *NetworkInterfaceSubresource) Read(l object.VirtualDeviceList) error {
	vd, err := r.FindVirtualDevice(l)
	if err != nil {
		return fmt.Errorf("cannot find network device: %s", err)
	}
	device, err := baseVirtualDeviceToBaseVirtualEthernetCard(vd)
	if err != nil {
		return err
	}

	// Determine the interface type, and set the field appropriately. As a fallback,
	// we actually set adapter_type here to "unknown" if we don't support the NIC
	// type, as we can determine all of the other settings without having to
	// worry about the adapter type, and on update, the adapter type will be
	// rectified by removing the existing NIC and replacing it with a new one.
	switch device.(type) {
	case *types.VirtualVmxnet3:
		r.Set("adapter_type", networkInterfaceSubresourceTypeVmxnet3)
	case *types.VirtualE1000:
		r.Set("adapter_type", networkInterfaceSubresourceTypeE1000)
	default:
		r.Set("adapter_type", networkInterfaceSubresourceTypeUnknown)
	}

	// The rest of the information we need to get by reading the attributes off
	// the base card object.
	card := device.GetVirtualEthernetCard()

	// Determine the network
	var netID string
	switch backing := card.Backing.(type) {
	case *types.VirtualEthernetCardNetworkBackingInfo:
		if backing.Network == nil {
			return fmt.Errorf("could not determine network information from NIC backing")
		}
		netID = backing.Network.Value
	case *types.VirtualEthernetCardOpaqueNetworkBackingInfo:
		onet, err := nsx.OpaqueNetworkFromNetworkID(r.client, backing.OpaqueNetworkId)
		if err != nil {
			return err
		}
		netID = onet.Reference().Value
	case *types.VirtualEthernetCardDistributedVirtualPortBackingInfo:
		pg, err := dvportgroup.FromKey(r.client, backing.Port.SwitchUuid, backing.Port.PortgroupKey)
		if err != nil {
			return err
		}
		netID = pg.Reference().Value
	default:
		return fmt.Errorf("unknown network interface backing %T", card.Backing)
	}
	r.Set("network_id", netID)

	if card.AddressType == string(types.VirtualEthernetCardMacTypeManual) {
		r.Set("use_static_mac", true)
	} else {
		r.Set("use_static_mac", false)
	}
	r.Set("mac_address", card.MacAddress)

	if card.ResourceAllocation != nil {
		r.Set("bandwidth_limit", card.ResourceAllocation.Limit)
		r.Set("bandwidth_reservation", card.ResourceAllocation.Reservation)
		r.Set("bandwidth_share_count", card.ResourceAllocation.Share.Shares)
		r.Set("bandwidth_share_level", card.ResourceAllocation.Share.Level)
	}

	// Save the device key and address data
	ctlr, err := findControllerForDevice(l, vd)
	if err != nil {
		return err
	}
	r.SaveDevIDs(vd, ctlr)
	return nil
}

// Update updates a vsphere_virtual_machine network_interface sub-resource.
func (r *NetworkInterfaceSubresource) Update(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	vd, err := r.FindVirtualDevice(l)
	if err != nil {
		return nil, fmt.Errorf("cannot find network device: %s", err)
	}
	device, err := baseVirtualDeviceToBaseVirtualEthernetCard(vd)
	if err != nil {
		return nil, err
	}

	// We maintain the final update spec in place, versus just the simple device
	// list, to support deletion of virtual devices so that they can replaced by
	// ones with different device types.
	var spec []types.BaseVirtualDeviceConfigSpec

	// A change in device_type is essentially a ForceNew. We would normally veto
	// this, but network devices are not extremely mission critical if they go
	// away, so we can support in-place modification of them in configuration by
	// just pushing a delete of the old device and adding a new version of the
	// device, with the old device unit number preserved so that it (hopefully)
	// gets the same device position as its previous incarnation, allowing old
	// device aliases to work, etc.
	if r.HasChange("device_type") {
		card := device.GetVirtualEthernetCard()
		newDevice, err := l.CreateEthernetCard(r.Get("adapter_type").(string), card.Backing)
		if err != nil {
			return nil, err
		}
		newCard := newDevice.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard()
		// Copy controller attributes and unit number
		newCard.ControllerKey = card.ControllerKey
		if card.UnitNumber != nil {
			var un int32
			un = *card.UnitNumber
			newCard.UnitNumber = &un
		}
		// Ensure the device starts connected
		// Set the key
		newCard.Key = l.NewKey()
		// Push the delete of the old device
		bvd := baseVirtualEthernetCardToBaseVirtualDevice(device)
		dspec, err := object.VirtualDeviceList{bvd}.ConfigSpec(types.VirtualDeviceConfigSpecOperationRemove)
		if err != nil {
			return nil, err
		}
		spec = append(spec, dspec...)
		// new device now becomes the old device and we proceed with the rest
		device, err = baseVirtualDeviceToBaseVirtualEthernetCard(newDevice)
		if err != nil {
			return nil, err
		}
	}

	card := device.GetVirtualEthernetCard()
	if r.HasChange("use_static_mac") {
		if r.Get("use_static_mac").(bool) {
			card.AddressType = string(types.VirtualEthernetCardMacTypeManual)
			card.MacAddress = r.Get("mac_address").(string)
		} else {
			// If we've gone from a static MAC address to a auto-generated one, we need
			// to check what address type we need to set things to.
			if r.client.ServiceContent.About.ApiType != "VirtualCenter" {
				// ESXi - type is "generated"
				card.AddressType = string(types.VirtualEthernetCardMacTypeGenerated)
			} else {
				// vCenter - type is "assigned"
				card.AddressType = string(types.VirtualEthernetCardMacTypeAssigned)
			}
			card.MacAddress = ""
		}
	}
	alloc := &types.VirtualEthernetCardResourceAllocation{
		Limit:       structure.Int64Ptr(int64(r.Get("bandwidth_limit").(int))),
		Reservation: structure.Int64Ptr(int64(r.Get("bandwidth_reservation").(int))),
		Share: types.SharesInfo{
			Shares: int32(r.Get("bandwidth_share_count").(int)),
			Level:  types.SharesLevel(r.Get("bandwidth_share_level").(string)),
		},
	}
	card.ResourceAllocation = alloc

	var op types.VirtualDeviceConfigSpecOperation
	if card.Key < 0 {
		// Negative key means that we are re-creating this device
		op = types.VirtualDeviceConfigSpecOperationAdd
	} else {
		op = types.VirtualDeviceConfigSpecOperationEdit
	}

	bvd := baseVirtualEthernetCardToBaseVirtualDevice(device)
	uspec, err := object.VirtualDeviceList{bvd}.ConfigSpec(op)
	if err != nil {
		return nil, err
	}
	spec = append(spec, uspec...)
	return spec, nil
}

// Delete deletes a vsphere_virtual_machine network_interface sub-resource.
func (r *NetworkInterfaceSubresource) Delete(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	vd, err := r.FindVirtualDevice(l)
	if err != nil {
		return nil, fmt.Errorf("cannot find network device: %s", err)
	}
	device, err := baseVirtualDeviceToBaseVirtualEthernetCard(vd)
	if err != nil {
		return nil, err
	}
	bvd := baseVirtualEthernetCardToBaseVirtualDevice(device)
	spec, err := object.VirtualDeviceList{bvd}.ConfigSpec(types.VirtualDeviceConfigSpecOperationRemove)
	if err != nil {
		return nil, err
	}
	return spec, nil
}

// assignEthernetCard is a subset of the logic that goes into AssignController
// right now but with an unit offset of 7. This is based on what we have
// observed on vSphere in terms of reserved PCI unit numbers (the first NIC
// automatically gets re-assigned to unit number 7 if it's not that already.)
func (r *NetworkInterfaceSubresource) assignEthernetCard(l object.VirtualDeviceList, device types.BaseVirtualDevice, c types.BaseVirtualController) error {
	// The PCI device offset. This seems to be where vSphere starts assigning
	// virtual NICs on the PCI controller.
	pciDeviceOffset := int32(7)

	// The first part of this is basically the private newUnitNumber function
	// from VirtualDeviceList, with a maximum unit count of 10. This basically
	// means that no more than 10 virtual NICs can be assigned right now, which
	// hopefully should be plenty.
	units := make([]bool, 10)

	ckey := c.GetVirtualController().Key

	for _, device := range l {
		d := device.GetVirtualDevice()
		if d.ControllerKey != ckey || d.UnitNumber == nil || *d.UnitNumber < pciDeviceOffset || *d.UnitNumber >= pciDeviceOffset+10 {
			continue
		}
		units[*d.UnitNumber-pciDeviceOffset] = true
	}

	// Now that we know which units are used, we can pick one
	newUnit := int32(r.Index) + pciDeviceOffset
	if units[newUnit] {
		return fmt.Errorf("device unit at %d is currently in use on the PCI bus", newUnit)
	}

	d := device.GetVirtualDevice()
	d.ControllerKey = c.GetVirtualController().Key
	d.UnitNumber = &newUnit
	if d.Key == 0 {
		d.Key = -1
	}
	return nil
}
