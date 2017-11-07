package virtualdevice

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

const (
	subresourceTypeDisk             = "disk"
	subresourceTypeNetworkInterface = "network_interface"
	subresourceTypeCdrom            = "cdrom"
)

const (
	// SubresourceControllerTypeIDE is a string representation of IDE controller
	// classes.
	SubresourceControllerTypeIDE = "ide"

	// SubresourceControllerTypeSCSI is a string representation of all SCSI
	// controller types.
	//
	// This is mainly used when computing IDs so that we can use a more general
	// device search.
	SubresourceControllerTypeSCSI = "scsi"

	// SubresourceControllerTypeParaVirtual is a string representation of the
	// VMware PV SCSI controller type.
	SubresourceControllerTypeParaVirtual = "pvscsi"

	// SubresourceControllerTypeLsiLogicSAS is a string representation of the
	// LSI Logic virtual SCSI controller type.
	SubresourceControllerTypeLsiLogicSAS = "lsilogic-sas"

	// SubresourceControllerTypePCI is a string representation of PCI controller
	// classes.
	SubresourceControllerTypePCI = "pci"
)

const (
	subresourceControllerTypeMixed   = "mixed"
	subresourceControllerTypeUnknown = "unknown"
)

var subresourceIDControllerTypeAllowedValues = []string{
	SubresourceControllerTypeIDE,
	SubresourceControllerTypeSCSI,
	SubresourceControllerTypePCI,
}

var sharesLevelAllowedValues = []string{
	string(types.SharesLevelLow),
	string(types.SharesLevelNormal),
	string(types.SharesLevelHigh),
	string(types.SharesLevelCustom),
}

// SCSIBusTypeAllowedValues exports the currently list of SCSI controller types
// that we support in the resource. The user is only allowed to select a type
// in this list, which should be used in a ValidateFunc on the appropriate
// field.
var SCSIBusTypeAllowedValues = []string{
	SubresourceControllerTypeParaVirtual,
	SubresourceControllerTypeLsiLogicSAS,
}

// newSubresourceFunc is a method signature for the wrapper methods that create
// a new instance of a specific subresource  that is derived from the base
// subresoruce object. It's used in the general apply and read operation
// methods, which themselves are called usually from higher-level apply
// functions for virtual devices.
type newSubresourceFunc func(*govmomi.Client, int, int, *schema.ResourceData) SubresourceInstance

// SubresourceInstance is an interface for derivative objects of Subresoruce.
// It's used on the general apply and read operation methods, and contains both
// exported methods of the base Subresource type and the CRUD methods that
// should be supplied by derivative objects.
//
// Note that this interface should be used sparingly - as such, only the
// methods that are needed by inparticular functions external to most virtual
// device workflows are exported into this interface.
type SubresourceInstance interface {
	Create(object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error)
	Read(object.VirtualDeviceList) error
	Update(object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error)
	Delete(object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error)

	DevAddr() string
	Addr() string
	Set(string, interface{}) error
	Schema() map[string]*schema.Schema
	State() map[string]interface{}
}

// controllerTypeToClass converts a controller type to a specific short-form
// controller class, namely for use with working with IDs.
func controllerTypeToClass(c types.BaseVirtualController) string {
	switch c.(type) {
	case *types.VirtualIDEController:
		return SubresourceControllerTypeIDE
	case *types.VirtualPCIController:
		return SubresourceControllerTypePCI
	case *types.ParaVirtualSCSIController, *types.VirtualBusLogicController,
		*types.VirtualLsiLogicController, *types.VirtualLsiLogicSASController:
		return SubresourceControllerTypeSCSI
	}
	panic(fmt.Errorf("unsupported controller type %T", c))
}

// Subresource defines a common interface for device sub-resources in the
// vsphere_virtual_machine resource.
//
// This object is designed to be used by parts of the resource with workflows
// that are so complex in their own right that probably the only way to handle
// their management is to treat them like resources themselves.
//
// This structure of this resource loosely follows schema.Resource with having
// CRUD and maintaining a set of resource data to work off of. However, since
// we are using schema.Resource, we take some liberties that we normally would
// not be able to take, or need to take considering the context of the data we
// are working with.
//
// Inparticular functions implement this structure by creating an instance into
// it, much like how a resource creates itself by creating an instance of
// schema.Resource.
type Subresource struct {
	// The index of this subresource - should either be an index or hash. It's up
	// to the upstream object to set this to something useful.
	Index int

	// The resource schema. This is an internal field as we build on this field
	// later on with common keys for all subresources, namely the internal ID.
	schema map[string]*schema.Schema

	// The client connection.
	client *govmomi.Client

	// The subresource type. This should match the key that the subresource is
	// named in the schema, such as "disk" or "network_interface".
	srtype string

	// The resource data - this should be loaded when the resource is created.
	data map[string]interface{}

	// The old resource data, if it exists.
	olddata map[string]interface{}

	// The root-level ResourceData for this resource. This should be used
	// sparingly. All ResourceData-style calls in this object do not use this
	// data, save for flagging reboot.
	resourceData *schema.ResourceData

	// The root-level ResourceDiff for this resource. Like resourceDiff, this
	// should be used sparingly and is not available in any calls save those that
	// come in through CustomizeDiff paths. Customization of actual sub-resource
	// diffs should happen against the entire set in higher-level functions.
	resourceDiff *schema.ResourceDiff
}

// subresourceSchema is a map[string]*schema.Schema of common schema fields.
// This includes the internal_id field, which is used as a unique ID for the
// lifecycle of this resource.
func subresourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"key": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The unique device ID for this device within its virtual machine.",
		},
		"device_address": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The internally-computed address of this device, such as scsi:0:1, denoting scsi bus #0 and device unit 1.",
		},
	}
}

// Addr returns the resource address for this subresource.
func (r *Subresource) Addr() string {
	return fmt.Sprintf("%s.%d", r.srtype, r.Index)
}

// Get hands off to r.data.Get, with an address relative to this subresource.
func (r *Subresource) Get(key string) interface{} {
	return r.data[key]
}

// Set sets the specified key/value pair in the subresource.
func (r *Subresource) Set(key string, value interface{}) {
	if v := structure.NormalizeValue(value); v != nil {
		r.data[key] = v
	}
}

// HasChange checks to see if there has been a change in the resource data
// since the last update.
//
// Note that this operation may only be useful during update operations,
// depending on subresource-specific workflow.
func (r *Subresource) HasChange(key string) bool {
	o, n := r.GetChange(key)
	return !reflect.DeepEqual(o, n)
}

// GetChange gets the old and new values for the value specified by key.
func (r *Subresource) GetChange(key string) (interface{}, interface{}) {
	new := r.data[key]
	// No old data means no change,  so we use the new value as a placeholder.
	old := r.data[key]
	if r.olddata != nil {
		old = r.olddata[key]
	}
	return old, new
}

// GetWithRestart checks to see if a field has been modified, returns the new
// value, and sets restart if it has changed.
func (r *Subresource) GetWithRestart(key string) interface{} {
	if r.HasChange(key) {
		r.SetRestart(key)
	}
	return r.Get(key)
}

// GetWithVeto returns the value specified by key, but returns an error if it
// has changed. The intention here is to block changes to the resource in a
// fashion that would otherwise result in forcing a new resource.
func (r *Subresource) GetWithVeto(key string) (interface{}, error) {
	if r.HasChange(key) {
		return r.Get(key), fmt.Errorf("cannot change the value of %q - must delete and re-create device", key)
	}
	return r.Get(key), nil
}

// SetRestart sets reboot_required in the global ResourceData. The key is only
// required for logging.
func (r *Subresource) SetRestart(key string) {
	log.Printf("[DEBUG] %s: Resource argument %q requires a VM restart", r, key)
	r.resourceData.Set("reboot_required", true)
}

// Data returns the underlying data map.
func (r *Subresource) Data() map[string]interface{} {
	return r.data
}

// Hash calculates a set hash for the current data. If you want a hash for
// error reporting a device address, it's probably a good idea to run this at
// the beginning of a run as any set calls will change the value this
// ultimately calculates.
func (r *Subresource) Hash() int {
	hf := schema.HashResource(&schema.Resource{Schema: r.schema})
	return hf(r.data)
}

// computeDevAddr handles the logic for saveAddr and allows it to be used
// outside of a subresource.
func computeDevAddr(device types.BaseVirtualDevice, ctlr types.BaseVirtualController) string {
	vd := device.GetVirtualDevice()
	vc := ctlr.GetVirtualController()
	ctype := controllerTypeToClass(ctlr)
	parts := []string{
		ctype,
		strconv.Itoa(int(vc.BusNumber)),
		strconv.Itoa(int(structure.DeRef(vd.UnitNumber).(int32))),
	}
	return strings.Join(parts, ":")
}

// SaveDevIDs saves the device's current key, and also the device_address. The
// latter is a computed schema field that contains the controller type, the
// controller's bus number, and the device's unit number on that controller.
// This helps locate the device when the key is in flux (such as when devices
// are just being created).
func (r *Subresource) SaveDevIDs(device types.BaseVirtualDevice, ctlr types.BaseVirtualController) {
	r.Set("key", device.GetVirtualDevice().Key)
	r.Set("device_address", computeDevAddr(device, ctlr))
}

// DevAddr returns the device_address attribute in the subresource. This
// function exists mainly as a functional counterpart to SaveDevIDs.
func (r *Subresource) DevAddr() string {
	return r.Get("device_address").(string)
}

// splitDevAddr splits an device addres into its inparticular parts and asserts
// that we have all the correct data.
func splitDevAddr(id string) (string, int, int, error) {
	parts := strings.Split(id, ":")
	if len(parts) < 3 {
		return "", 0, 0, fmt.Errorf("invalid ID %q", id)
	}
	ct, cbs, dus := parts[0], parts[1], parts[2]
	cb, cbe := strconv.Atoi(cbs)
	du, due := strconv.Atoi(dus)
	var found bool
	for _, v := range subresourceIDControllerTypeAllowedValues {
		if v == ct {
			found = true
		}
	}
	if !found {
		return ct, cb, du, fmt.Errorf("invalid controller type %q found in ID", ct)
	}
	if cbe != nil {
		return ct, cb, du, fmt.Errorf("invalid bus number %q found in ID", cbs)
	}
	if due != nil {
		return ct, cb, du, fmt.Errorf("invalid disk unit number %q found in ID", dus)
	}
	return ct, cb, du, nil
}

// findVirtualDeviceInListControllerSelectFunc returns a function that can be
// used with VirtualDeviceList.Select to locate a controller device based on
// the criteria that we have laid out.
func findVirtualDeviceInListControllerSelectFunc(ct string, cb int) func(types.BaseVirtualDevice) bool {
	return func(device types.BaseVirtualDevice) bool {
		var ctlr types.BaseVirtualController
		switch ct {
		case SubresourceControllerTypeIDE:
			if v, ok := device.(*types.VirtualIDEController); ok {
				ctlr = v
				goto controllerFound
			}
			return false
		case SubresourceControllerTypeSCSI:
			switch v := device.(type) {
			case *types.ParaVirtualSCSIController:
				ctlr = v
				goto controllerFound
			case *types.VirtualLsiLogicSASController:
				ctlr = v
				goto controllerFound
			}
			return false
		case SubresourceControllerTypePCI:
			if v, ok := device.(*types.VirtualPCIController); ok {
				ctlr = v
				goto controllerFound
			}
			return false
		}
	controllerFound:
		vc := ctlr.GetVirtualController()
		if vc.BusNumber == int32(cb) {
			return true
		}
		return false
	}
}

// findVirtualDeviceInListDeviceSelectFunc returns a function that can be used
// with VirtualDeviceList.Select to locate a virtual device based on its
// controller device key, and the unit number on the device.
func findVirtualDeviceInListDeviceSelectFunc(ckey int32, du int) func(types.BaseVirtualDevice) bool {
	return func(d types.BaseVirtualDevice) bool {
		vd := d.GetVirtualDevice()
		if vd.ControllerKey == ckey && vd.UnitNumber != nil && *vd.UnitNumber == int32(du) {
			return true
		}
		return false
	}
}

// findControllerForDevice locates a controller via its virtual device.
func findControllerForDevice(l object.VirtualDeviceList, bvd types.BaseVirtualDevice) (types.BaseVirtualController, error) {
	vd := bvd.GetVirtualDevice()
	ctlr := l.FindByKey(vd.ControllerKey)

	if ctlr == nil {
		return nil, fmt.Errorf("could not find controller key %d for device %d", vd.ControllerKey, vd.Key)
	}

	return ctlr.(types.BaseVirtualController), nil
}

// FindVirtualDeviceByAddr locates the subresource's virtual device in the
// supplied VirtualDeviceList by its device address.
func (r *Subresource) FindVirtualDeviceByAddr(l object.VirtualDeviceList) (types.BaseVirtualDevice, error) {
	log.Printf("[DEBUG] FindVirtualDevice: Looking for device with address %s", r.DevAddr())
	ct, cb, du, err := splitDevAddr(r.DevAddr())
	if err != nil {
		return nil, err
	}

	// find the controller
	csf := findVirtualDeviceInListControllerSelectFunc(ct, cb)
	ctlrs := l.Select(csf)
	if len(ctlrs) != 1 {
		return nil, fmt.Errorf("invalid controller result - %d results returned (expected 1): type %q, bus number: %d", len(ctlrs), ct, cb)
	}
	ctlr := ctlrs[0]

	// find the device
	ckey := ctlr.GetVirtualDevice().Key
	dsf := findVirtualDeviceInListDeviceSelectFunc(ckey, du)
	devices := l.Select(dsf)
	if len(devices) != 1 {
		return nil, fmt.Errorf("invalid device result - %d results returned (expected 1): controller key %q, disk number: %d", len(devices), ckey, du)
	}
	device := devices[0]
	log.Printf("[DEBUG] FindVirtualDevice: Device found: %s", l.Name(device))
	return device, nil
}

// FindVirtualDevice will attempt to find an address by its device key if it is
// > 0, otherwise it will attempt to locate it by its device address.
func (r *Subresource) FindVirtualDevice(l object.VirtualDeviceList) (types.BaseVirtualDevice, error) {
	if key := r.Get("key").(int); key > 0 {
		log.Printf("[DEBUG] FindVirtualDevice: Looking for device with key %d", key)
		if dev := l.FindByKey(int32(key)); dev != nil {
			log.Printf("[DEBUG] FindVirtualDevice: Device found: %s", l.Name(dev))
			return dev, nil
		}
		return nil, fmt.Errorf("could not find device with key %d", key)
	}
	return r.FindVirtualDeviceByAddr(l)
}

// String prints out the device sub-resource's information including the ID at
// time of instantiation, the short name of the disk, and the current device
// key and address.
func (r *Subresource) String() string {
	devaddr := r.Get("device_address").(string)
	if devaddr == "" {
		devaddr = "<new device>"
	}
	return fmt.Sprintf("%s (key %d at %s)", r.Addr(), r.Get("key").(int), devaddr)
}

// swapSCSIDevice swaps out the supplied controller for a new one of the
// supplied controller type. Any connected devices are re-connected at the same
// device units on the new device. A list of changes is returned.
func swapSCSIDevice(l object.VirtualDeviceList, device types.BaseVirtualSCSIController, ct string) ([]types.BaseVirtualDeviceConfigSpec, error) {
	log.Printf("[DEBUG] swapSCSIDevice: Swapping SCSI device for one of controller type %s: %s", ct, l.Name(device.(types.BaseVirtualDevice)))
	var spec []types.BaseVirtualDeviceConfigSpec
	bvd := device.(types.BaseVirtualDevice)
	cspec, err := object.VirtualDeviceList{bvd}.ConfigSpec(types.VirtualDeviceConfigSpecOperationRemove)
	if err != nil {
		return nil, err
	}
	spec = append(spec, cspec...)

	nsd, err := l.CreateSCSIController(ct)
	if err != nil {
		return nil, err
	}
	cspec, err = object.VirtualDeviceList{nsd}.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
	if err != nil {
		return nil, err
	}
	spec = append(spec, cspec...)
	ockey := device.GetVirtualSCSIController().Key
	nckey := nsd.GetVirtualDevice().Key
	for _, vd := range l {
		if vd.GetVirtualDevice().Key == ockey {
			vd.GetVirtualDevice().Key = nckey
			cspec, err := object.VirtualDeviceList{vd}.ConfigSpec(types.VirtualDeviceConfigSpecOperationEdit)
			if err != nil {
				return nil, err
			}
			spec = append(spec, cspec...)
		}
	}
	log.Printf("[DEBUG] swapSCSIDevice: Outgoing device config spec: %s", DeviceChangeString(spec))
	return spec, nil
}

// NormalizeSCSIBus checks the SCSI controllers on the virtual machine and
// either creates them if they don't exist, or migrates them to the specified
// controller type. Devices are migrated to the new controller appropriately. A
// spec slice is returned with the changes.
//
// All 4 slots on the SCSI bus are normalized to the appropriate device.
func NormalizeSCSIBus(l object.VirtualDeviceList, ct string) (object.VirtualDeviceList, []types.BaseVirtualDeviceConfigSpec, error) {
	log.Printf("[DEBUG] NormalizeSCSIBus: Normalizing SCSI bus to device type %s", ct)
	var spec []types.BaseVirtualDeviceConfigSpec
	ctlrs := make([]types.BaseVirtualSCSIController, 4)
	// Don't worry about doing any fancy select stuff here, just go thru the
	// VirtualDeviceList and populate the controllers.
	for _, dev := range l {
		if sc, ok := dev.(types.BaseVirtualSCSIController); ok {
			ctlrs[sc.GetVirtualSCSIController().BusNumber] = sc
		}
	}
	log.Printf("[DEBUG] NormalizeSCSIBus: Current SCSI bus contents: %s", scsiControllerListString(ctlrs))
	// Now iterate over the controllers
	for n, ctlr := range ctlrs {
		if ctlr == nil {
			log.Printf("[DEBUG] NormalizeSCSIBus: Creating SCSI controller of type %s at bus number %d", ct, n)
			nc, err := l.CreateSCSIController(ct)
			if err != nil {
				return nil, nil, err
			}
			cspec, err := object.VirtualDeviceList{nc}.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
			if err != nil {
				return nil, nil, err
			}
			spec = append(spec, cspec...)
			l = applyDeviceChange(l, cspec)
			continue
		}
		if l.Type(ctlr.(types.BaseVirtualDevice)) == ct {
			continue
		}
		cspec, err := swapSCSIDevice(l, ctlr, ct)
		if err != nil {
			return nil, nil, err
		}
		spec = append(spec, cspec...)
		l = applyDeviceChange(l, cspec)
		continue
	}
	log.Printf("[DEBUG] NormalizeSCSIBus: Outgoing device list: %s", DeviceListString(l))
	log.Printf("[DEBUG] NormalizeSCSIBus: Outgoing device config spec: %s", DeviceChangeString(spec))
	return l, spec, nil
}

// ReadSCSIBusState checks the SCSI bus state and returns a device type
// depending on if all controllers are one specific kind or not.
func ReadSCSIBusState(l object.VirtualDeviceList) string {
	ctlrs := make([]types.BaseVirtualSCSIController, 4)
	for _, dev := range l {
		if sc, ok := dev.(types.BaseVirtualSCSIController); ok {
			ctlrs[sc.GetVirtualSCSIController().BusNumber] = sc
		}
	}
	log.Printf("[DEBUG] ReadSCSIBusState: SCSI controller layout: %s", scsiControllerListString(ctlrs))
	if ctlrs[0] == nil {
		return subresourceControllerTypeUnknown
	}
	last := l.Type(ctlrs[0].(types.BaseVirtualDevice))
	for _, ctlr := range ctlrs[1:] {
		if ctlr == nil || l.Type(ctlr.(types.BaseVirtualDevice)) != last {
			return subresourceControllerTypeMixed
		}
	}
	return last
}

// getSCSIController picks a SCSI controller at the specific bus number supplied.
func pickSCSIController(l object.VirtualDeviceList, bus int) (types.BaseVirtualController, error) {
	log.Printf("[DEBUG] pickSCSIController: Looking for SCSI controller at bus number %d", bus)
	l = l.Select(func(device types.BaseVirtualDevice) bool {
		switch d := device.(type) {
		case types.BaseVirtualSCSIController:
			return d.GetVirtualSCSIController().BusNumber == int32(bus)
		}
		return false
	})

	if len(l) == 0 {
		return nil, fmt.Errorf("could not find scsi controller at bus number %d", bus)
	}

	log.Printf("[DEBUG] pickSCSIController: Found SCSI controller: %s", l.Name(l[0]))
	return l[0].(types.BaseVirtualController), nil
}

// ControllerForCreateUpdate wraps the controller selection logic to make it
// easier to use in create or update operations. If the controller type is a
// SCSI device, the bus number is searched as well.
func (r *Subresource) ControllerForCreateUpdate(l object.VirtualDeviceList, ct string, bus int) (types.BaseVirtualController, error) {
	log.Printf("[DEBUG] ControllerForCreateUpdate: Looking for controller type %s", ct)
	var ctlr types.BaseVirtualController
	var err error
	switch ct {
	case SubresourceControllerTypeIDE:
		ctlr = l.PickController(&types.VirtualIDEController{})
	case SubresourceControllerTypeSCSI:
		ctlr, err = pickSCSIController(l, bus)
	case SubresourceControllerTypePCI:
		ctlr = l.PickController(&types.VirtualPCIController{})
	default:
		return nil, fmt.Errorf("invalid controller type %T", ct)
	}
	if err != nil {
		return nil, err
	}
	if ctlr == nil {
		return nil, fmt.Errorf("could not find an available %s controller", ct)
	}

	// Assert that we are on bus 0 when we aren't looking for a SCSI controller.
	// We currently do not support attaching devices to multiple non-SCSI buses.
	if ctlr.GetVirtualController().BusNumber != 0 && ct != SubresourceControllerTypeSCSI {
		return nil, fmt.Errorf("there are no available slots on the primary %s controller", ct)
	}
	log.Printf("[DEBUG] ControllerForCreateUpdate: Found controller: %s", l.Name(ctlr.(types.BaseVirtualDevice)))

	return ctlr, nil
}

// applyDeviceChange applies a pending types.BaseVirtualDeviceConfigSpec to a
// working set to either add, remove, or update devices so that the working
// VirtualDeviceList is as up to date as possible.
func applyDeviceChange(l object.VirtualDeviceList, cs []types.BaseVirtualDeviceConfigSpec) object.VirtualDeviceList {
	log.Printf("[DEBUG] applyDeviceChange: Applying changes: %s", DeviceChangeString(cs))
	log.Printf("[DEBUG] applyDeviceChange: Device list before changes: %s", DeviceListString(l))
	for _, s := range cs {
		spec := s.GetVirtualDeviceConfigSpec()
		switch spec.Operation {
		case types.VirtualDeviceConfigSpecOperationAdd:
			l = append(l, spec.Device)
		case types.VirtualDeviceConfigSpecOperationEdit:
			// Edit operations may not be 100% necessary to apply. This is because
			// more often than not, the device will probably be edited in place off
			// of the original reference, meaning that the slice should actually
			// point to the updated item. However, the safer of the two options is to
			// assume that this may *not* be happening as we are not enforcing that
			// in implementation anywhere.
			for n, dev := range l {
				if dev.GetVirtualDevice().Key == spec.Device.GetVirtualDevice().Key {
					l[n] = spec.Device
				}
			}
		case types.VirtualDeviceConfigSpecOperationRemove:
			for i := 0; i < len(l); i++ {
				dev := l[i]
				if dev.GetVirtualDevice().Key == spec.Device.GetVirtualDevice().Key {
					l = append(l[:i], l[i+1:]...)
					i--
				}
			}
		default:
			panic("unknown op")
		}
	}
	log.Printf("[DEBUG] applyDeviceChange: Device list after changes: %s", DeviceListString(l))
	return l
}

// DeviceListString pretty-prints each device in a virtual device list, used
// for logging purposes and what not.
func DeviceListString(l object.VirtualDeviceList) string {
	var names []string
	for _, d := range l {
		if d == nil {
			names = append(names, "<nil>")
		} else {
			names = append(names, l.Name(d))
		}
	}
	return strings.Join(names, ",")
}

// DeviceChangeString pretty-prints a slice of VirtualDeviceConfigSpec.
func DeviceChangeString(specs []types.BaseVirtualDeviceConfigSpec) string {
	var strs []string
	for _, v := range specs {
		spec := v.GetVirtualDeviceConfigSpec()
		strs = append(strs, fmt.Sprintf("(%s: %T at key %d)", string(spec.Operation), spec.Device, spec.Device.GetVirtualDevice().Key))
	}
	return strings.Join(strs, ",")
}

// subresourceListString takes a list of sub-resources and pretty-prints the
// key and device address.
func subresourceListString(data []interface{}) string {
	var strs []string
	for _, v := range data {
		if v == nil {
			strs = append(strs, "(<nil>)")
			continue
		}
		m := v.(map[string]interface{})
		devaddr := m["device_address"].(string)
		if devaddr == "" {
			devaddr = "<new device>"
		}
		strs = append(strs, fmt.Sprintf("(key %d at %s)", m["key"].(int), devaddr))
	}
	return strings.Join(strs, ",")
}

// scsiControllerListString pretty-prints a slice of SCSI controllers.
func scsiControllerListString(ctlrs []types.BaseVirtualSCSIController) string {
	var l object.VirtualDeviceList
	for _, ctlr := range ctlrs {
		if ctlr == nil {
			l = append(l, types.BaseVirtualDevice(nil))
		} else {
			l = append(l, ctlr.(types.BaseVirtualDevice))
		}
	}
	return DeviceListString(l)
}
