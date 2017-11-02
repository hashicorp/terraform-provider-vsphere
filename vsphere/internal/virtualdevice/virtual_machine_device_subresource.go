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

// orpahnedDeviceMinIndex is the index that we start adding orphaned devices
// at.
const orpahnedDeviceMinIndex = 1000

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
	// The resource schema. This is an internal field as we build on this field
	// later on with common keys for all subresources, namely the internal ID.
	schema map[string]*schema.Schema

	// The client connection.
	client *govmomi.Client

	// The subresource type. This should match the key that the subresource is
	// named in the schema, such as "disk" or "network_interface".
	srtype string

	// The subresource index.
	index int

	// The old subresource index in the event of sets - this allows us to give
	// accurate figures to HasChange and GetChange.
	oldindex int

	// A layer of keys set by this subresource during its apply or refresh run,
	// used in the event of sets.
	setdata map[string]interface{}

	// An instance pointing to the entire live resource's ResourceData. We use
	// the type and index data to extrapolate the sub-resource's data.
	data *schema.ResourceData
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

// Schema returns the schema for this subresource. The internal schema is
// merged with some common keys, which includes the internal_id field which is
// used as a unique ID for the lifecycle of this resource.
func (r *Subresource) Schema() map[string]*schema.Schema {
	s := r.schema
	structure.MergeSchema(s, subresourceSchema())
	return s
}

// Addr returns the address of this specific subresource.
func (r *Subresource) Addr() string {
	addr, _ := r.addrs()
	return addr
}

func (r *Subresource) addrs() (string, string) {
	return fmt.Sprintf("%s.%d", r.srtype, r.index), fmt.Sprintf("%s.%d", r.srtype, r.oldindex)
}

// keyAddrs computes the relative address of this specific subresource in the
// full ResourceData set.
func (r *Subresource) keyAddrs(k string) (string, string) {
	old, new := r.addrs()
	return fmt.Sprintf("%s.%s", old, k), fmt.Sprintf("%s.%s", new, k)
}

// Get hands off to r.data.Get, with an address relative to this subresource.
func (r *Subresource) Get(key string) interface{} {
	if r.setdata != nil {
		if v, ok := r.setdata[key]; ok {
			return v
		}
	}
	addr, _ := r.keyAddrs(key)
	return r.data.Get(addr)
}

// Set sets the specified key/value pair in the subresource.
//
// Only full lists or sets can be set in ResourceData right now - so we can't
// actually do something like set "disk.10.key" directly. However, to simulate
// this, we just simply read the set, set the key we need, and then write it
// back out. This allows us to abstract this away from the consumer.
//
// Note that right now this probably only works with primitives in the root
// of the sub-resource, which is fine as there are no implementations planned
// that would require nested fields in the sub-resources, however if in the
// future this changes this will need to be reviewed.
func (r *Subresource) Set(key string, value interface{}) error {
	log.Printf("[TRACE] r.Set(): %s.%d.%s: %#v", r.srtype, r.index, key, value)
	switch s := r.data.Get(r.srtype).(type) {
	case *schema.Set:
		log.Printf("[TRACE] r.Set(): %s.%d.%s: Resource is a set, must set in temporary layer", r.srtype, r.index, key)
		// Dealing with a set here, we need to keep track of this data separate.
		if r.setdata == nil {
			r.setdata = make(map[string]interface{})
		}
		r.setdata[key] = structure.DeRef(value)
		return nil
	case []interface{}:
		m := s[r.index].(map[string]interface{})
		m[key] = structure.DeRef(value)
		err := r.data.Set(r.srtype, s)
		if err != nil {
			log.Printf("[DEBUG] Error updating parent subresource set for %s.%d.%s: %s", r.srtype, r.index, key, err)
		}
		return err
	}
	log.Printf("[DEBUG] %s: invalid sub-resource type", r.srtype)
	return fmt.Errorf("%s: invalid sub-resource type", r.srtype)
}

// HasChange checks to see if there has been a change in the resource data
// since the last update.
func (r *Subresource) HasChange(key string) bool {
	o, n := r.GetChange(key)
	return !reflect.DeepEqual(o, n)
}

// GetChange gets the old and new values for the value specified by key.
func (r *Subresource) GetChange(key string) (interface{}, interface{}) {
	oa, na := r.keyAddrs(key)
	if r.index == r.oldindex {
		return r.data.GetChange(na)
	}
	od := r.data.Get(oa)
	nd := r.Get(key)
	return od, nd
}

// GetWithRestart checks to see if a field has been modified, returns the new
// value, and sets restart if it has changed.
func (r *Subresource) GetWithRestart(key string) interface{} {
	if r.HasChange(key) {
		// VMware supports hot-add of virtual devices, so if this is a new
		// sub-resource, don't worry about rebooting.
		if r.Get("device_address") != "" {
			r.SetRestart()
		}
	}
	return r.Get(key)
}

// GetWithVeto returns the value specified by key, but returns an error if it
// has changed. The intention here is to block changes to the resource in a
// fashion that would otherwise result in forcing a new resource.
func (r *Subresource) GetWithVeto(key string) (interface{}, error) {
	if r.HasChange(key) {
		// Only veto updates, if internal_id is not set yet, this is a create
		// operation and should be allowed to go through.
		if r.Get("device_address") != "" {
			return r.Get(key), fmt.Errorf("cannot change the value of %q - must delete and re-create device", key)
		}
	}
	return r.Get(key), nil
}

// SetRestart sets reboot_required in the global ResourceData.
func (r *Subresource) SetRestart() error {
	return r.data.Set("reboot_required", true)
}

// State returns a map[string]interface{} data object for the subresource,
// rolling in any data set during the run.
func (r *Subresource) State() map[string]interface{} {
	d := r.data.Get(r.srtype)
	var m map[string]interface{}
	switch s := d.(type) {
	case []interface{}:
		m = s[r.index].(map[string]interface{})
	case *schema.Set:
		for _, v := range s.List() {
			hf := schema.HashResource(&schema.Resource{Schema: r.Schema()})
			if hf(v) == r.index {
				m = v.(map[string]interface{})
				for k, v := range r.setdata {
					m[k] = v
				}
			}
		}
	}
	return m
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

// SaveDevIDs saves the resource ID of the subresource to device_address. This
// is a computed schema field that contains the controller type, the
// controller's bus number, and the device's unit number on that controller.
//
// This is an ID internal to Terraform that helps us locate the resource later,
// as device keys are unfortunately volatile and can only really be relied on
// for a single operation, as such they are unsuitable for use to check a
// resource later on.
func (r *Subresource) SaveDevIDs(device types.BaseVirtualDevice, ctlr types.BaseVirtualController) {
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

// FindVirtualDeviceByAddr locates the subresource's virtual device in the
// supplied VirtualDeviceList by its device address.
func (r *Subresource) FindVirtualDeviceByAddr(l object.VirtualDeviceList) (types.BaseVirtualDevice, error) {
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
	return device, nil
}

// FindVirtualDevice will attempt to find an address by its device key if it is
// > 0, otherwise it will attempt to locate it by its device address.
func (r *Subresource) FindVirtualDevice(l object.VirtualDeviceList) (types.BaseVirtualDevice, error) {
	if key := r.Get("key").(int); key > 0 {
		if dev := l.FindByKey(int32(key)); dev != nil {
			return dev, nil
		}
		return nil, fmt.Errorf("could not find device with key %d", key)
	}
	return r.FindVirtualDeviceByAddr(l)
}

// swapSCSIDevice swaps out the supplied controller for a new one of the
// supplied controller type. Any connected devices are re-connected at the same
// device units on the new device. A list of changes is returned.
func swapSCSIDevice(l object.VirtualDeviceList, device types.BaseVirtualSCSIController, ct string) ([]types.BaseVirtualDeviceConfigSpec, error) {
	var spec []types.BaseVirtualDeviceConfigSpec
	nsd, err := l.CreateSCSIController(ct)
	if err != nil {
		return nil, err
	}
	cspec, err := object.VirtualDeviceList{nsd}.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
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
	// Put the deletion of the old device at the end of the list as I'm not too
	// sure how vSphere applies these changes. Better safe than sorry.
	bvd := device.(types.BaseVirtualDevice)
	cspec, err = object.VirtualDeviceList{bvd}.ConfigSpec(types.VirtualDeviceConfigSpecOperationRemove)
	if err != nil {
		return nil, err
	}
	spec = append(spec, cspec...)
	return spec, nil
}

// NormalizeSCSIBus checks the SCSI controllers on the virtual machine and
// either creates them if they don't exist, or migrates them to the specified
// controller type. Devices are migrated to the new controller appropriately. A
// spec slice is returned with the changes.
//
// All 4 slots on the SCSI bus are normalized to the appropriate device.
func NormalizeSCSIBus(l object.VirtualDeviceList, ct string) (object.VirtualDeviceList, []types.BaseVirtualDeviceConfigSpec, error) {
	var spec []types.BaseVirtualDeviceConfigSpec
	ctlrs := make([]types.BaseVirtualSCSIController, 4)
	// Don't worry about doing any fancy select stuff here, just go thru the
	// VirtualDeviceList and populate the controllers.
	for _, dev := range l {
		if sc, ok := dev.(types.BaseVirtualSCSIController); ok {
			ctlrs[sc.GetVirtualSCSIController().BusNumber] = sc
		}
	}
	// Now iterate over the controllers
	for _, ctlr := range ctlrs {
		if ctlr == nil {
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
		if l.Name(ctlr.(types.BaseVirtualDevice)) == ct {
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

	return l[0].(types.BaseVirtualController), nil
}

// ControllerForCreateUpdate wraps the controller selection logic to make it
// easier to use in create or update operations. If the controller type is a
// SCSI device, the bus number is searched as well.
func (r *Subresource) ControllerForCreateUpdate(l object.VirtualDeviceList, ct string, bus int) (types.BaseVirtualController, error) {
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

	return ctlr, nil
}

// applyDeviceChange applies a pending types.BaseVirtualDeviceConfigSpec to a
// working set to either add, remove, or update devices so that the working
// VirtualDeviceList is as up to date as possible.
func applyDeviceChange(l object.VirtualDeviceList, cs []types.BaseVirtualDeviceConfigSpec) object.VirtualDeviceList {
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
			for n, dev := range l {
				if dev.GetVirtualDevice().Key == spec.Device.GetVirtualDevice().Key {
					l = append(l[:n], l[n+1:]...)
				}
			}
		default:
			panic("unknown op")
		}
	}
	return l
}

// getDeviceChangeSet returns an old and new set for the device subresource
// type based on the subresource schema type, as we allow both sets and lists
// to be subresource types. For sets, the intersection is discarded so that
// no-ops are not unnecessarily checked.
func getDeviceChangeSet(d *schema.ResourceData, srtype string) ([]interface{}, []interface{}) {
	o, n := d.GetChange(srtype)
	switch n.(type) {
	case []interface{}:
		return o.([]interface{}), n.([]interface{})
	case *schema.Set:
		// Make an intersection set. Any device in the intersection is a device
		// that is not changing, so we ignore those.
		ids := o.(*schema.Set).Intersection(n.(*schema.Set))
		ods := o.(*schema.Set).Difference(ids)
		nds := n.(*schema.Set).Difference(ids)
		return ods.List(), nds.List()
	}
	panic(fmt.Errorf("unsupported sub-resource type %T", n))
}

// indexOrHash either returns the index passed into it, or a set hash,
// depending on the subresource type schema.
func indexOrHash(d *schema.ResourceData, srtype string, f newSubresourceFunc, idx int, current, old interface{}) (int, int) {
	switch d.Get(srtype).(type) {
	case []interface{}:
		// Noop pretty much
		return idx, idx
	case *schema.Set:
		hf := schema.HashResource(&schema.Resource{Schema: f(nil, 0, 0, nil).Schema()})
		ch := hf(current)
		oh := ch
		if old != nil {
			oh = hf(old)
		}
		return ch, oh
	}
	panic(fmt.Errorf("unsupported sub-resource type %T", d.Get(srtype)))
}

// deviceApplyOperation processes an apply operation for a specific device
// class.
//
// The function takes the root resource's ResourceData, the provider
// connection, and the device list as known to vSphere at the start of this
// operation. All disk operations are carried out, with both the complete,
// updated, VirtualDeviceList, and the complete list of changes returned as a
// slice of BaseVirtualDeviceConfigSpec.
//
// This is a helper that should be exposed via a higher-level resource type.
func deviceApplyOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList, srtype string, newResourceFunc newSubresourceFunc) (object.VirtualDeviceList, []types.BaseVirtualDeviceConfigSpec, error) {
	ods, nds := getDeviceChangeSet(d, srtype)

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
		// The device was not found in the new set, which means this is a destroy.
		idx, _ := indexOrHash(d, srtype, newResourceFunc, n, om, nil)
		r := newResourceFunc(c, idx, idx, d)
		dspec, err := r.Delete(l)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		l = applyDeviceChange(l, dspec)
		spec = append(spec, dspec...)
	}

	// Now check for creates and updates.
	// The results of this operation are committed to state after the operation
	// completes.
	var updates []interface{}
nextNew:
	for n, ne := range nds {
		nm := ne.(map[string]interface{})
		for _, oe := range ods {
			om := oe.(map[string]interface{})
			if nm["key"] == om["key"] {
				// This is an update
				idx, oidx := indexOrHash(d, srtype, newResourceFunc, n, nm, om)
				r := newResourceFunc(c, idx, oidx, d)
				uspec, err := r.Update(l)
				if err != nil {
					return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
				}
				l = applyDeviceChange(l, uspec)
				spec = append(spec, uspec...)
				updates = append(updates, r.State())
				continue nextNew
			}
		}
		// Device not found in old set, this is a new subresource.
		idx, _ := indexOrHash(d, srtype, newResourceFunc, n, nm, nil)
		r := newResourceFunc(c, idx, idx, d)
		cspec, err := r.Create(l)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		l = applyDeviceChange(l, cspec)
		spec = append(spec, cspec...)
		updates = append(updates, r.State())
	}

	// We are now done! Return the updated device list and config spec. Save updates as well.
	if err := d.Set(srtype, updates); err != nil {
		return nil, nil, err
	}
	return l, spec, nil
}

// isVirtualDisk returns true if the type in question is a virtual disk.
func isVirtualDisk(a interface{}) bool {
	_, ok := a.(*types.VirtualDisk)
	return ok
}

// isNetworkInterface returns true if the type in question is a network
// interface.
func isNetworkInterface(a interface{}) bool {
	return reflect.TypeOf(a).Implements(reflect.TypeOf((*types.BaseVirtualEthernetCard)(nil)).Elem())
}

// isVirtualCdrom returns true if the type in question is a virtual disk.
func isVirtualCdrom(a interface{}) bool {
	_, ok := a.(*types.VirtualCdrom)
	return ok
}

// deviceRefreshOperation handles refreshes of device sub-resources. It also
// gathers any device of a specific class that is not accounted for and creates
// sub-resource instances for them in state - these will be naturally culled by
// the next apply operation.
//
// As this is purely a read-only operation except for relation to state, only
// errors are returned.
func deviceRefreshOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList, srtype string, newResourceFunc newSubresourceFunc) error {
	// Go over the device list, looking for devices we support. We use srtype to
	// determine what kind of type we are looking for.
	var eligibleDevice func(interface{}) bool
	switch srtype {
	case subresourceTypeDisk:
		eligibleDevice = isVirtualDisk
	case subresourceTypeNetworkInterface:
		eligibleDevice = isNetworkInterface
	case subresourceTypeCdrom:
		eligibleDevice = isVirtualCdrom
	default:
		return fmt.Errorf("invalid subresource type %s. This is bug with Terraform and should be reported", srtype)
	}
	devices := l.Select(func(device types.BaseVirtualDevice) bool {
		if eligibleDevice(device) {
			return true
		}
		return false
	})
	curSet, _ := getDeviceChangeSet(d, srtype)
	var newSet []interface{}
	// First check for negative keys. These are freshly added devices that are
	// usually coming into read post-create.
	for n, item := range curSet {
		m := item.(map[string]interface{})
		if m["key"].(int) < 1 {
			idx, _ := indexOrHash(d, srtype, newResourceFunc, n, m, nil)
			r := newResourceFunc(c, idx, idx, d)
			if err := r.Read(l); err != nil {
				return fmt.Errorf("%s: %s", r.Addr(), err)
			}
			newM := r.State()
			if newM["key"].(int) < 1 {
				// This should not have happened - if it did, our device
				// creation/update logic failed somehow that we were not able to track.
				return fmt.Errorf("device %v with address %v still unaccounted for after update/read", newM["key"], newM["device_address"])
			}
			newSet = append(newSet, r.State())
			for i := 0; i < len(devices); i++ {
				device := devices[i]
				if device.GetVirtualDevice().Key == int32(newM["key"].(int)) {
					devices = append(devices[:i], devices[i+1:]...)
				}
			}
		}
	}

	// Go over the remaining devices, refresh via key, and then remove their entries as well.
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
			idx, _ := indexOrHash(d, srtype, newResourceFunc, n, m, nil)
			r := newResourceFunc(c, idx, idx, d)
			if err := r.Read(l); err != nil {
				return fmt.Errorf("%s: %s", r.Addr(), err)
			}
			// Done reading, push this onto our new set and remove the device from the list
			newSet = append(newSet, r.State())
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
		idx, _ := indexOrHash(d, srtype, newResourceFunc, orpahnedDeviceMinIndex+n, m, nil)
		r := newResourceFunc(c, idx, idx, d)
		r.Set("key", m["key"])
		r.Set("device_address", m["device_address"])
		if err := r.Read(l); err != nil {
			return fmt.Errorf("%s: %s", r.Addr(), err)
		}
		newSet = append(newSet, r.State())
	}

	return d.Set(srtype, newSet)
}
