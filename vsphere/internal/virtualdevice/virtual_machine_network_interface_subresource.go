package virtualdevice

import (
	"context"
	"fmt"

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
var networkInterfaceSubresourceSchema = map[string]*schema.Schema{
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
	"key": {
		Type:        schema.TypeInt,
		Computed:    true,
		Description: "The unique device ID for this device within the virtual machine configuration.",
	},
}

// NetworkInterfaceSubresource represents a vsphere_virtual_machine
// network_interface sub-resource, with a complex device lifecycle.
type NetworkInterfaceSubresource struct {
	*Subresource
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
	ctlr, cspec, err := r.ControllerForCreateUpdate(l, SubresourceControllerTypePCI)
	if err != nil {
		return nil, err
	}
	if len(cspec) > 0 {
		// We don't support adding new PCI devices right now, but this is here just
		// in case, and for consistency with other resources.
		l = append(l, cspec[0].GetVirtualDeviceConfigSpec().Device)
		spec = append(spec, cspec...)
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
	l.AssignController(device, ctlr)
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
	r.SaveID(device, ctlr)
	dspec, err := object.VirtualDeviceList{device}.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
	if err != nil {
		return nil, err
	}
	spec = append(spec, dspec...)
	return spec, nil
}

// Read reads a vsphere_virtual_machine network_interface sub-resource.
func (r *NetworkInterfaceSubresource) Read(l object.VirtualDeviceList, client *govmomi.Client) error {
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
		onet, err := nsx.OpaqueNetworkFromNetworkID(client, backing.OpaqueNetworkId)
		if err != nil {
			return err
		}
		netID = onet.Reference().Value
	case *types.VirtualEthernetCardDistributedVirtualPortBackingInfo:
		pg, err := dvportgroup.FromKey(client, backing.Port.SwitchUuid, backing.Port.PortgroupKey)
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

	// Save the device key
	r.Set("key", card.Key)
	return nil
}

// Update updates a vsphere_virtual_machine network_interface sub-resource.
func (r *NetworkInterfaceSubresource) Update(l object.VirtualDeviceList, client *govmomi.Client) ([]types.BaseVirtualDeviceConfigSpec, error) {
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
			if client.ServiceContent.About.ApiType != "VirtualCenter" {
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
