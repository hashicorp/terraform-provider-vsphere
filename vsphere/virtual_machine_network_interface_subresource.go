package vsphere

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

const (
	resourceVSphereVirtualMachineNetworkInterfaceTypeE1000   = "e1000"
	resourceVSphereVirtualMachineNetworkInterfaceTypeVmxnet3 = "vmxnet3"
	resourceVSphereVirtualMachineNetworkInterfaceTypeUnknown = "unknown"
)

var resourceVSphereVirtualMachineNetworkInterfaceTypeAllowedValues = []string{
	resourceVSphereVirtualMachineNetworkInterfaceTypeE1000,
	resourceVSphereVirtualMachineNetworkInterfaceTypeVmxnet3,
}

var resourceVSphereVirtualMachineNetworkInterfaceMACAddressTypeAllowedValues = []string{
	string(types.VirtualEthernetCardMacTypeManual),
}

// resourceVSphereVirtualMachineNetworkInterface defines a
// vsphere_virtual_machine network_interface sub-resource.
//
// The workflow here is CRUD-like, and designed to be portable to other uses in
// the future, however various changes are made to the interface to account for
// the fact that this is not necessarily a fully-fledged resource in its own
// right.
type resourceVSphereVirtualMachineNetworkInterface struct {
	// The old resource data.
	oldData map[string]interface{}

	// The new resource data.
	newData map[string]interface{}

	// This is flagged if anything in the CRUD process requires a VM restart. The
	// parent CRUD is responsible for flagging the appropriate information and
	// doing the necessary restart before applying the resulting ConfigSpec.
	restart bool
}

// resourceVSphereVirtualMachineNetworkInterfaceSchema returns the schema for the disk
// sub-resource.
func resourceVSphereVirtualMachineNetworkInterfaceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
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
			Default:      resourceVSphereVirtualMachineNetworkInterfaceTypeE1000,
			Description:  "The controller type. Can be one of e1000 or vmxnet3.",
			ValidateFunc: validation.StringInSlice(resourceVSphereVirtualMachineNetworkInterfaceTypeAllowedValues, false),
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
		"controller_bus_number": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The bus index for the PCI controller that this device is attached to.",
		},
		"controller_key": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The unique device ID for the PCI controller this device is attached to.",
		},
		"unit_number": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The unit number of the device on the PCI controller.",
		},
		"key": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The unique device ID for this device within the virtual machine configuration.",
		},
		"internal_id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The internally-computed ID of this resource, local to Terraform - this is controller_bus_number:unit_number.",
		},
	}
}

// pickPCIController returns the PCI device in the VirtualDeviceList.
//
// Adding new devices is currently not supported, but we are keeping this a
// function to add this functionality in the future if we need it.
func pickOrCreatePCIController(l object.VirtualDeviceList) (types.BaseVirtualController, error) {
	ctlr := l.PickController(&types.VirtualPCIController{})
	if ctlr == nil {
		// This should hopefully never, never happen. ;)
		return nil, fmt.Errorf("PCI controller not found on this virtual machine")
	}
	return ctlr, nil
}

// get gets the field from the new set of resource data.
func (r *resourceVSphereVirtualMachineNetworkInterface) get(key string) interface{} {
	return r.newData[key]
}

// set sets the field in the newData set.
func (r *resourceVSphereVirtualMachineNetworkInterface) set(key string, value interface{}) {
	r.newData[key] = deRef(value)
}

// hasChange checks to see if a field has been modified and returns true if it
// has.
func (r *resourceVSphereVirtualMachineNetworkInterface) hasChange(key string) bool {
	return r.oldData[key] != r.newData[key]
}

// getChange returns the old and new values for the supplied key.
func (r *resourceVSphereVirtualMachineNetworkInterface) getChange(key string) (interface{}, interface{}) {
	return r.oldData[key], r.newData[key]
}

// getWithRestart checks to see if a field has been modified, returns the new
// value, and sets restart if it has changed.
func (r *resourceVSphereVirtualMachineNetworkInterface) getWithRestart(key string) interface{} {
	if r.hasChange(key) {
		r.restart = true
	}
	return r.get(key)
}

// getWithVeto returns the value specified by key, but returns an error if it
// has changed. The intention here is to block changes to the resource in a
// fashion that would otherwise result in forcing a new resource.
func (r *resourceVSphereVirtualMachineNetworkInterface) getWithVeto(key string) (interface{}, error) {
	if r.hasChange(key) {
		// only veto updates, if internal_id is not set yet, this is a create
		// operation and should be allowed to go through.
		if r.get("internal_id") != "" {
			return r.get(key), fmt.Errorf("cannot change the value of %q - must delete and re-create device", key)
		}
	}
	return r.get(key), nil
}

// saveID saves the resource ID to internal_id. It also sets the computed
// values that it tracks.
//
// This is an ID internal to Terraform that helps us locate the resource later,
// as device keys are unfortunately volatile and can only really be relied on
// for a single operation, as such they are unsuitable for use to check a
// resource later on.
//
// The resource format is a combination of the controller type as supplied to
// the resource, the bus number of that controller, and the device number on
// the controller.
func (r *resourceVSphereVirtualMachineNetworkInterface) saveID(device types.BaseVirtualEthernetCard, ctlr types.BaseVirtualController) {
	vc := ctlr.GetVirtualController()
	card := device.GetVirtualEthernetCard()
	parts := []string{
		strconv.Itoa(int(vc.BusNumber)),
		strconv.Itoa(int(deRef(card.UnitNumber).(int32))),
	}
	r.set("controller_bus_number", vc.BusNumber)
	r.set("controller_key", vc.Key)
	r.set("unit_number", card.UnitNumber)
	r.set("internal_id", strings.Join(parts, ":"))
}

// id returns the internal_id attribute in the subresource. This function
// exists mainly as a functional counterpart to saveID.
func (r *resourceVSphereVirtualMachineNetworkInterface) id() string {
	return r.get("internal_id").(string)
}

// splitVirtualMachineNetworkInterfaceID splits an ID into its inparticular parts
// and asserts that we have all the correct data.
func splitVirtualMachineNetworkInterfaceID(id string) (int, int, error) {
	parts := strings.Split(id, ":")
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("invalid ID %q", id)
	}
	cbs, dus := parts[0], parts[1]
	cb, cbe := strconv.Atoi(cbs)
	du, due := strconv.Atoi(dus)
	if cbe != nil {
		return cb, du, fmt.Errorf("invalid bus number %q found in ID", cbs)
	}
	if due != nil {
		return cb, du, fmt.Errorf("invalid disk unit number %q found in ID", dus)
	}
	return cb, du, nil
}

// findVirtualMachineNetworkInterfaceInListControllerSelectFunc returns a function
// that can be used with VirtualDeviceList.Select to locate a controller device
// based on the criteria that we have laid out.
//
// Note that this is mainly a placeholder to support the very remote
// possibility that there could be multiple PCI buses on a system.
func findVirtualMachineNetworkInterfaceInListControllerSelectFunc(cb int) func(types.BaseVirtualDevice) bool {
	return func(device types.BaseVirtualDevice) bool {
		if v, ok := device.(*types.VirtualPCIController); ok {
			vc := v.GetVirtualController()
			if vc.BusNumber == int32(cb) {
				return true
			}
		}
		return false
	}
}

// findVirtualMachineNetworkInterfaceInListNetworkInterfaceSelectFunc returns a
// function that can be used with VirtualDeviceList.Select to locate a disk
// device based on its controller device key, and the disk number on the
// device.
func findVirtualMachineNetworkInterfaceInListNetworkInterfaceSelectFunc(ckey int32, du int) func(types.BaseVirtualDevice) bool {
	return func(device types.BaseVirtualDevice) bool {
		nic, ok := device.(types.BaseVirtualEthernetCard)
		if !ok {
			return false
		}
		card := nic.GetVirtualEthernetCard()
		if card.ControllerKey == ckey && card.UnitNumber != nil && *card.UnitNumber == int32(du) {
			return true
		}
		return false
	}
}

// findVirtualMachineNetworkInterfaceInList looks for a specific device in the
// device list given a specific disk device key. nil is returned if no device
// is found.
func findVirtualMachineNetworkInterfaceInList(l object.VirtualDeviceList, id string) (types.BaseVirtualEthernetCard, error) {
	cb, du, err := splitVirtualMachineNetworkInterfaceID(id)
	if err != nil {
		return nil, err
	}

	// find the controller
	csf := findVirtualMachineNetworkInterfaceInListControllerSelectFunc(cb)
	ctlrs := l.Select(csf)
	if len(ctlrs) != 1 {
		return nil, fmt.Errorf("invalid controller result - %d results returned (expected 1): bus number: %d", len(ctlrs), cb)
	}
	ctlr := ctlrs[0]

	// find the NIC
	ckey := ctlr.GetVirtualDevice().Key
	dsf := findVirtualMachineNetworkInterfaceInListNetworkInterfaceSelectFunc(ckey, du)
	nics := l.Select(dsf)
	if len(nics) != 1 {
		return nil, fmt.Errorf("invalid NIC result - %d results returned (expected 1): controller key %q, disk number: %d", len(nics), ckey, du)
	}
	nic := nics[0]
	return nic.(types.BaseVirtualEthernetCard), nil
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
func baseVirtualDeviceToBaseVirtualEthernetCard(v types.BaseVirtualDevice) types.BaseVirtualEthernetCard {
	switch t := v.(type) {
	case *types.VirtualE1000:
		return types.BaseVirtualEthernetCard(t)
	case *types.VirtualE1000e:
		return types.BaseVirtualEthernetCard(t)
	case *types.VirtualPCNet32:
		return types.BaseVirtualEthernetCard(t)
	case *types.VirtualSriovEthernetCard:
		return types.BaseVirtualEthernetCard(t)
	case *types.VirtualVmxnet2:
		return types.BaseVirtualEthernetCard(t)
	case *types.VirtualVmxnet3:
		return types.BaseVirtualEthernetCard(t)
	}
	panic(fmt.Errorf("unknown ethernet card type %T", v))
}

// controllerForCreateUpdate wraps the controller selection logic mainly used
// for creation so that we can re-use it in Update.
//
// If the controller is new, it's returned as the second return value, as a
// VirtualDeviceList, for easy appending to outbound devices and the working
// set.
func (r *resourceVSphereVirtualMachineNetworkInterface) controllerForCreateUpdate(l object.VirtualDeviceList) (types.BaseVirtualController, object.VirtualDeviceList, error) {
	var newDevices object.VirtualDeviceList
	ctlr, err := pickOrCreatePCIController(l)
	if err != nil {
		return nil, nil, err
	}

	// Is this a new controller? If so, we need to push this to our working
	// device set so that its device key is accounted for, in addition to the
	// list of new devices that we are returning as part of the device creation,
	// so that they can be added to the ConfigSpec properly.
	if ctlr.GetVirtualController().Key < 0 {
		newDevices = append(newDevices, ctlr.(*types.VirtualPCIController))
	}
	return ctlr, newDevices, nil
}

// Create creates a vsphere_virtual_machine network_interface sub-resource.
func (r *resourceVSphereVirtualMachineNetworkInterface) Create(l object.VirtualDeviceList, client *govmomi.Client) ([]types.BaseVirtualDeviceConfigSpec, error) {
	var newDevices object.VirtualDeviceList
	var ctlr types.BaseVirtualController
	ctlr, ncl, err := r.controllerForCreateUpdate(l)
	if err != nil {
		return nil, err
	}
	l = append(l, ncl...)
	newDevices = append(newDevices, ncl...)

	// govmomi has helpers that allow the easy fetching of a network's backing
	// info, once we actually know what that backing is. Set all of that stuff up
	// now.
	net, err := networkFromID(client, r.get("network_id").(string))
	if err != nil {
		return nil, err
	}
	bctx, bcancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer bcancel()
	backing, err := net.EthernetCardBackingInfo(bctx)
	if err != nil {
		return nil, err
	}
	device, err := l.CreateEthernetCard(r.get("adapter_type").(string), backing)
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
	if r.get("use_static_mac").(bool) {
		card.AddressType = string(types.VirtualEthernetCardMacTypeManual)
		card.MacAddress = r.get("mac_address").(string)
	}
	alloc := &types.VirtualEthernetCardResourceAllocation{
		Limit:       int64Ptr(int64(r.get("bandwidth_limit").(int))),
		Reservation: int64Ptr(int64(r.get("bandwidth_reservation").(int))),
		Share: types.SharesInfo{
			Shares: int32(r.get("bandwidth_share_count").(int)),
			Level:  types.SharesLevel(r.get("bandwidth_share_level").(string)),
		},
	}
	card.ResourceAllocation = alloc

	// Done here. Save ID, push the device to the new device list and return.
	r.saveID(device.(types.BaseVirtualEthernetCard), ctlr)
	newDevices = append(newDevices, device)
	return newDevices.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
}

// Read reads a vsphere_virtual_machine network_interface sub-resource and
// commits the data to the newData layer.
func (r *resourceVSphereVirtualMachineNetworkInterface) Read(l object.VirtualDeviceList, client *govmomi.Client) error {
	id := r.id()
	device, err := findVirtualMachineNetworkInterfaceInList(l, id)
	if err != nil {
		return fmt.Errorf("cannot find network device: %s", err)
	}
	// Determine the interface type, and set the field appropriately. As a fallback,
	// we actually set adapter_type here to "unknown" if we don't support the NIC
	// type, as we can determine all of the other settings without having to
	// worry about the adapter type, and on update, the adapter type will be
	// rectified by removing the existing NIC and replacing it with a new one.
	switch device.(type) {
	case *types.VirtualVmxnet3:
		r.set("adapter_type", resourceVSphereVirtualMachineNetworkInterfaceTypeVmxnet3)
	case *types.VirtualE1000:
		r.set("adapter_type", resourceVSphereVirtualMachineNetworkInterfaceTypeE1000)
	default:
		r.set("adapter_type", resourceVSphereVirtualMachineNetworkInterfaceTypeUnknown)
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
		onet, err := opaqueNetworkFromNetworkID(client, backing.OpaqueNetworkId)
		if err != nil {
			return err
		}
		netID = onet.Reference().Value
	case *types.VirtualEthernetCardDistributedVirtualPortBackingInfo:
		pg, err := dvPortgroupFromKey(client, backing.Port.SwitchUuid, backing.Port.PortgroupKey)
		if err != nil {
			return err
		}
		netID = pg.Reference().Value
	default:
		return fmt.Errorf("unknown network interface backing %T", card.Backing)
	}
	r.set("network_id", netID)

	if card.AddressType == string(types.VirtualEthernetCardMacTypeManual) {
		r.set("use_static_mac", true)
	} else {
		r.set("use_static_mac", false)
	}
	r.set("mac_address", card.MacAddress)

	if card.ResourceAllocation != nil {
		r.set("bandwidth_limit", card.ResourceAllocation.Limit)
		r.set("bandwidth_reservation", card.ResourceAllocation.Reservation)
		r.set("bandwidth_share_count", card.ResourceAllocation.Share.Shares)
		r.set("bandwidth_share_level", card.ResourceAllocation.Share.Level)
	}

	// Save the device key
	r.set("key", card.Key)
	return nil
}

// Update updates a vsphere_virtual_machine network_interface sub-resource.
func (r *resourceVSphereVirtualMachineNetworkInterface) Update(l object.VirtualDeviceList, client *govmomi.Client) ([]types.BaseVirtualDeviceConfigSpec, error) {
	id := r.id()
	device, err := findVirtualMachineNetworkInterfaceInList(l, id)
	if err != nil {
		return nil, fmt.Errorf("cannot find network device: %s", err)
	}

	// We maintain the final update spec in place, versus just the simple device
	// list, to support deletion of virtual devices so that they can replaced by
	// ones with different device types.
	var updateSpec []types.BaseVirtualDeviceConfigSpec

	// A change in device_type is essentially a ForceNew. We would normally veto
	// this, but network devices are not extremely mission critical if they go
	// away, so we can support in-place modification of them in configuration by
	// just pushing a delete of the old device and adding a new version of the
	// device, with the old device unit number preserved so that it (hopefully)
	// gets the same device position as its previous incarnation, allowing old
	// device aliases to work, etc.
	if r.hasChange("device_type") {
		card := device.GetVirtualEthernetCard()
		newDevice, err := l.CreateEthernetCard(r.get("adapter_type").(string), card.Backing)
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
		spec, err := object.VirtualDeviceList{bvd}.ConfigSpec(types.VirtualDeviceConfigSpecOperationRemove)
		if err != nil {
			return nil, err
		}
		updateSpec = append(updateSpec, spec...)
		// new device now becomes the old device and we proceed with the rest
		device = baseVirtualDeviceToBaseVirtualEthernetCard(newDevice)
	}

	card := device.GetVirtualEthernetCard()
	if r.hasChange("use_static_mac") {
		if r.get("use_static_mac").(bool) {
			card.AddressType = string(types.VirtualEthernetCardMacTypeManual)
			card.MacAddress = r.get("mac_address").(string)
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
		Limit:       int64Ptr(int64(r.get("bandwidth_limit").(int))),
		Reservation: int64Ptr(int64(r.get("bandwidth_reservation").(int))),
		Share: types.SharesInfo{
			Shares: int32(r.get("bandwidth_share_count").(int)),
			Level:  types.SharesLevel(r.get("bandwidth_share_level").(string)),
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
	spec, err := object.VirtualDeviceList{bvd}.ConfigSpec(op)
	if err != nil {
		return nil, err
	}
	updateSpec = append(updateSpec, spec...)
	return updateSpec, nil
}

// Delete deletes a vsphere_virtual_machine network_interface sub-resource.
func (r *resourceVSphereVirtualMachineNetworkInterface) Delete(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	id := r.id()
	device, err := findVirtualMachineNetworkInterfaceInList(l, id)
	if err != nil {
		return nil, fmt.Errorf("cannot find network device: %s", err)
	}
	bvd := baseVirtualEthernetCardToBaseVirtualDevice(device)
	deleteSpec, err := object.VirtualDeviceList{bvd}.ConfigSpec(types.VirtualDeviceConfigSpecOperationRemove)
	if err != nil {
		return nil, err
	}
	return deleteSpec, nil
}
