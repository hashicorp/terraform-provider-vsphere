package virtualdevice

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
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
	// VMware
	SubresourceControllerTypeLsiLogicSAS = "lsilogic-sas"

	// SubresourceControllerTypePCI is a string representation of PCI controller
	// classes.
	SubresourceControllerTypePCI = "pci"
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

// newSubresourceFunc is a method signature for the wrapper methods that create
// a new instance of a specific subresource  that is derived from the base
// subresoruce object. It's used in the general apply and read operation
// methods, which themselves are called usually from higher-level apply
// functions for virtual devices.
type newSubresourceFunc func(*govmomi.Client, int, *schema.ResourceData) SubresourceInstance

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

	ID() string
	Addr() string
	Set(string, interface{}) error
	Schema() map[string]*schema.Schema
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

	// An instance pointing to the entire live resource's ResourceData. We use
	// the type and index data to extrapolate the sub-resource's data.
	data *schema.ResourceData
}

// subresourceSchema is a map[string]*schema.Schema of common schema fields.
// This includes the internal_id field, which is used as a unique ID for the
// lifecycle of this resource.
func subresourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"index": {
			Type:         schema.TypeInt,
			Required:     true,
			Description:  "A unique index for this device within its class. This ID cannot be recycled until it has been unused for at least one Terraform run.",
			ValidateFunc: validation.IntBetween(0, orpahnedDeviceMinIndex-1),
		},
		"internal_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The internally-computed ID of this resource, local to Terraform - this is controller_type:controller_bus_number:unit_number.",
		},
	}
}

// SubresourceHashFunc returns the value of index as the ID for an inparticular
// resource in its set.
//
// A Subresource is designed to be implemented as a *schema.Set to minimize the
// risk that configuration drift has against its values. Using a simple index
// value, while not protecting 100% against drift (this value could still be
// modified) or other problems (it's possible for someone to use duplicate
// values here, causing additional declarations to be ignored), it's better
// than most options available.
func SubresourceHashFunc(v interface{}) int {
	m := v.(map[string]interface{})
	return m["index"].(int)
}

// ValidateRegistry takes a map[string]interface{} designed to be a index ->
// internal_id key/value store. This is an extra safeguard against config drift
// - it enforces that index reuse doesn't happen over a single run, requiring
// the user do at least one run with a index removed before re-using it.
//
// An error is returned if there a set element has an index that does not match
// its internal_id.
func ValidateRegistry(registry map[string]interface{}, set *schema.Set) error {
	for _, v := range set.List() {
		m := v.(map[string]interface{})
		idx := strconv.Itoa(m["index"].(int))
		iid := m["internal_id"].(string)
		if rid, ok := registry[idx]; ok {
			if iid != rid {
				return fmt.Errorf("ID mismatch at index %s - old: %q new: %q. Please apply with index removed before re-using", idx, rid, iid)
			}
		}
	}
	return nil
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
	return fmt.Sprintf("%s.%d", r.srtype, r.index)
}

// keyAddr computes the relative address of this specific subresource in the
// full ResourceData set.
func (r *Subresource) keyAddr(k string) string {
	return fmt.Sprintf("%s.%s", r.Addr(), k)
}

// Get hands off to r.data.Get, with an address relative to this subresource.
func (r *Subresource) Get(key string) interface{} {
	return r.data.Get(r.keyAddr(key))
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
	// Block setting of the index value through this function.
	if key == "index" {
		errstr := fmt.Sprintf("%s.%d.%s: Setting of index key not allowed", r.srtype, r.index, key)
		log.Printf("[DEBUG] %s", errstr)
		return errors.New(errstr)
	}
	log.Printf("[TRACE] r.Set(): %s.%d.%s: %#v", r.srtype, r.index, key, value)
	s := r.data.Get(r.srtype).(*schema.Set)
	var found bool
	for _, v := range s.List() {
		m := v.(map[string]interface{})
		if m["index"].(int) == r.index {
			m[key] = structure.DeRef(value)
			found = true
		}
	}
	if !found {
		// We don't have data for this resource, so we need to add a brand new
		// element.
		m := map[string]interface{}{
			"index": r.index,
			key:     value,
		}
		s.Add(m)
	}
	err := r.data.Set(r.srtype, s)
	if err != nil {
		log.Printf("[DEBUG] Error updating parent subresource set for %s.%d.%s: %s", r.srtype, r.index, key, err)
	}
	return err
}

// HasChange hands off to r.data.HasChange, with an address relative to this
// subresource.
func (r *Subresource) HasChange(key string) bool {
	return r.data.HasChange(r.keyAddr(key))
}

// GetChange hands off to r.data.GetChange, with an address relative to this
// subresource.
func (r *Subresource) GetChange(key string) (interface{}, interface{}) {
	return r.data.GetChange(r.keyAddr(key))
}

// GetWithRestart checks to see if a field has been modified, returns the new
// value, and sets restart if it has changed.
func (r *Subresource) GetWithRestart(key string) interface{} {
	if r.HasChange(key) {
		// VMware supports hot-add of virtual devices, so if this is a new
		// sub-resource, don't worry about rebooting.
		if r.Get("internal_id") != "" {
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
		if r.Get("internal_id") != "" {
			return r.Get(key), fmt.Errorf("cannot change the value of %q - must delete and re-create device", key)
		}
	}
	return r.Get(key), nil
}

// SetRestart sets reboot_required in the global ResourceData.
func (r *Subresource) SetRestart() error {
	return r.data.Set("reboot_required", true)
}

// computeID handles the logic for saveID and allows it to be used outside of a
// subresource.
func computeID(device types.BaseVirtualDevice, ctlr types.BaseVirtualController) string {
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

// SaveID saves the resource ID of the subresource to internal_id. This is a
// computed schema field that contains the controller type, the controller's
// bus number, and the device's unit number on that controller.
//
// This is an ID internal to Terraform that helps us locate the resource later,
// as device keys are unfortunately volatile and can only really be relied on
// for a single operation, as such they are unsuitable for use to check a
// resource later on.
func (r *Subresource) SaveID(device types.BaseVirtualDevice, ctlr types.BaseVirtualController) {
	r.Set("internal_id", computeID(device, ctlr))
}

// ID returns the internal_id attribute in the subresource. This function
// exists mainly as a functional counterpart to SaveID.
func (r *Subresource) ID() string {
	return r.Get("internal_id").(string)
}

// splitInternalID splits an ID into its inparticular parts and asserts that we
// have all the correct data.
func splitInternalID(id string) (string, int, int, error) {
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

// FindVirtualDevice locates the subresource's virtual device in the supplied
// VirtualDeviceList, working off of the resource's internal_id attribute.
func (r *Subresource) FindVirtualDevice(l object.VirtualDeviceList) (types.BaseVirtualDevice, error) {
	ct, cb, du, err := splitInternalID(r.ID())
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

// pickOrCreateDiskController either finds a device of a specific type with an
// available slot, or creates a new one. An error is returned if there is a
// problem anywhere in this process or not possible.
//
// Note that this does not push the controller to the device list - this is
// done outside of this function, to keep things atomic at the end.
func pickOrCreateDiskController(l object.VirtualDeviceList, kind types.BaseVirtualController) (types.BaseVirtualController, error) {
	ctlr := l.PickController(kind)
	if ctlr == nil {
		var nc types.BaseVirtualDevice
		var err error
		switch kind.(type) {
		case *types.VirtualIDEController:
			nc, err = l.CreateIDEController()
			ctlr = nc.(*types.VirtualIDEController)
		case *types.ParaVirtualSCSIController:
			nc, err = l.CreateSCSIController(SubresourceControllerTypeParaVirtual)
			ctlr = nc.(*types.ParaVirtualSCSIController)
		case *types.VirtualLsiLogicSASController:
			nc, err = l.CreateSCSIController(SubresourceControllerTypeLsiLogicSAS)
			ctlr = nc.(*types.VirtualLsiLogicSASController)
		default:
			return nil, fmt.Errorf("cannot create new controller of type: %T", kind)
		}
		if err != nil {
			return nil, err
		}
	}
	return ctlr, nil
}

// ControllerForCreateUpdate wraps the controller selection logic to make it
// easier to use in create or update operations.
//
// If the controller is new, it's returned as the second return value, as a Add
// device change operation, for easy appending to outbound devices and the
// working set. Otherwise the device list is empty.
func (r *Subresource) ControllerForCreateUpdate(l object.VirtualDeviceList, ct string) (types.BaseVirtualController, []types.BaseVirtualDeviceConfigSpec, error) {
	var ctlr types.BaseVirtualController
	var err error
	switch ct {
	case SubresourceControllerTypeIDE:
		ctlr, err = pickOrCreateDiskController(l, &types.VirtualIDEController{})
	case SubresourceControllerTypeParaVirtual:
		ctlr, err = pickOrCreateDiskController(l, &types.ParaVirtualSCSIController{})
	case SubresourceControllerTypeLsiLogicSAS:
		ctlr, err = pickOrCreateDiskController(l, &types.VirtualLsiLogicSASController{})
	case SubresourceControllerTypePCI:
		ctlr, err = pickOrCreateDiskController(l, &types.VirtualPCIController{})
	default:
		return nil, nil, fmt.Errorf("invalid controller type %T", ct)
	}
	if err != nil {
		return nil, nil, err
	}

	// Is this a new controller? If so, we need to push this to our working
	// device set so that its device key is accounted for, in addition to the
	// list of new devices that we are returning as part of the device creation,
	// so that they can be added to the ConfigSpec properly.
	//
	// New controllers will have a negative device key and will *not* be in our
	// current device list. The former is more important to vSphere, but the
	// latter means we have already added a deviceChange spec for the controller
	// more than likely.
	var dl object.VirtualDeviceList
	var cs []types.BaseVirtualDeviceConfigSpec
	var inl bool
	for _, d := range l {
		if d.GetVirtualDevice().Key == ctlr.GetVirtualController().Key {
			inl = true
		}
	}
	if ctlr.GetVirtualController().Key < 0 && !inl {
		switch ct := ctlr.(type) {
		case *types.VirtualIDEController:
			dl = append(dl, ct)
		case *types.ParaVirtualSCSIController:
			dl = append(dl, ct)
		case *types.VirtualLsiLogicSASController:
			dl = append(dl, ct)
		default:
			// This should never happen, as if we don't support the controller for
			// creation, then a graceful error will be returned earlier in the logic
			// chain, so panic here.
			panic(fmt.Errorf("unhandled new controller type %T", ctlr))
		}
		var err error
		cs, err = dl.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
		if err != nil {
			// If there was some sort of issue generating the ConfigSpec I don't
			// think there's anything that the user can do to really rectify this.
			// Just panic here, as there is probably something really wrong.
			panic(err)
		}
	}
	return ctlr, cs, nil
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
			// or the original reference, meaning that the slice should actually
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
	// Fetch the ID registry for the device's resources. This should be a
	// map[int]string, but since TypeMap only supports map[string]string and
	// shows up as a map[string]interface{}. We make do.
	registry := d.Get(fmt.Sprintf("%s_internal_ids", srtype)).(map[string]interface{})
	if registry == nil {
		// Possibly dealing with a new VM resource.
		registry = make(map[string]interface{})
	}
	o, n := d.GetChange(srtype)
	ods := o.(*schema.Set)
	nds := n.(*schema.Set)
	// Validate against our new disk set, to make sure that there isn't any index
	// drift.
	if err := ValidateRegistry(registry, nds); err != nil {
		return nil, nil, err
	}

	// Make an intersection set. These are disks that we need to check for
	// changes later, but ones that are not in the intersection are either being
	// created or deleted.
	ids := ods.Intersection(nds)
	ods = ods.Difference(ids)
	nds = nds.Difference(ids)

	var spec []types.BaseVirtualDeviceConfigSpec

	// Our old and new sets now have an accurate description of devices that may
	// have been added or removed. Look for removed devices first.
	for _, oe := range ods.List() {
		m := oe.(map[string]interface{})
		r := newResourceFunc(c, m["index"].(int), d)
		dspec, err := r.Delete(l)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		l = applyDeviceChange(l, dspec)
		spec = append(spec, dspec...)
		// Delete the item from the registry if it exists.
		idx := strconv.Itoa(m["index"].(int))
		delete(registry, idx)
	}

	// Now create
	for _, ne := range nds.List() {
		m := ne.(map[string]interface{})
		r := newResourceFunc(c, m["index"].(int), d)
		cspec, err := r.Create(l)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		l = applyDeviceChange(l, cspec)
		spec = append(spec, cspec...)
		// Add the item to the registry with its ID, which will have been set as
		// part of the create process.
		idx := strconv.Itoa(m["index"].(int))
		registry[idx] = r.ID()
	}

	// Finally process any pending updates. We actually do a HasChange on the
	// direct address here to make sure we need to update in the first place.
	for _, ie := range ids.List() {
		m := ie.(map[string]interface{})
		r := newResourceFunc(c, m["index"].(int), d)
		if d.HasChange(r.Addr()) {
			uspec, err := r.Update(l)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
			}
			l = applyDeviceChange(l, uspec)
			spec = append(spec, uspec...)
			// The ID may have changed as part of this process, so save the ID just
			// in case.
			idx := strconv.Itoa(m["index"].(int))
			registry[idx] = r.ID()
		}
	}

	// Save the registry now, as there will have been changes to IDs that we
	// don't keep track of during read necessarily.
	if err := d.Set(fmt.Sprintf("%s_internal_ids", srtype), registry); err != nil {
		return nil, nil, fmt.Errorf("error committing ID registry: %s", err)
	}

	// We are now done! Return the updated device list and config spec.
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
	// Start the orphaned device counter at the minimum index value so that we
	// can add orphaned devices and OOB devices to state if need be.
	orphanedIndex := orpahnedDeviceMinIndex

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
	iids := d.Get(fmt.Sprintf("%s_internal_ids", srtype)).(map[string]interface{})
nextDevice:
	for _, bvd := range l {
		if !eligibleDevice(bvd) {
			// Quick guard to just skip devices that we aren't looking for this run
			continue
		}
		// Check to see if this is a device we are tracking in the ID registry.
		vd := bvd.GetVirtualDevice()
		var bvc types.BaseVirtualController
		for _, bvd := range l {
			if vd.ControllerKey == bvd.GetVirtualDevice().Key {
				bvc = reflect.ValueOf(bvd).Interface().(types.BaseVirtualController)
			}
		}
		ida := computeID(bvd, bvc)
		for k, v := range iids {
			idb := v.(string)
			if ida == idb {
				// We have a match of a device we are tracking in configuration. Read
				// the state for this resource and move on to the next device.
				idx, err := strconv.Atoi(k)
				if err != nil {
					return fmt.Errorf("bad index %q found in %s_internal_ids. This is a bug in Terraform and should be reported", k, srtype)
				}
				r := newResourceFunc(c, idx, d)
				if err := r.Read(l); err != nil {
					return fmt.Errorf("%s: %s", r.Addr(), err)
				}
				// Remove this ID from our working set of diffs. We use the remainder
				// as a list of set items to cull later.
				delete(iids, k)
				continue nextDevice
			}
		}
		// If we have searched the entire ID registry and have not found the device
		// we are looking for, then the device was more than likely added out of
		// band. We save this device to state at an index in a keyspace outside of
		// our normal resources, and save the internal ID and index to state as
		// well (things that don't happen in read).
		r := newResourceFunc(c, orphanedIndex, d)
		r.Set("internal_id", ida)
		if err := r.Read(l); err != nil {
			return fmt.Errorf("%s error reading orphaned device: %s", r.Addr(), err)
		}
		// Should be done here.
	}
	// If there were any items that have disappeared for any reason, we want to
	// make every effort to remove those items from our state, so that the next
	// diff is as accurate as possible as to what needs to happen to fix it.
	//
	// We do this by checking the remainder of our internal ID map and removing
	// items from our set and internal ID registry respectively.
	devsOld := d.Get(srtype).(*schema.Set)
	devsNew := d.Get(srtype).(*schema.Set)
	niids := d.Get(fmt.Sprintf("%s_internal_ids", srtype)).(map[string]interface{})
	for k := range iids {
		for _, v := range devsOld.List() {
			m := v.(map[string]interface{})
			if strconv.Itoa(m["index"].(int)) == k {
				devsNew.Remove(m)
				delete(niids, k)
			}
		}
	}
	if err := d.Set(fmt.Sprintf("%s_internal_ids", srtype), niids); err != nil {
		return fmt.Errorf("error saving new %s_internal_ids on refresh: %s", srtype, err)
	}
	return d.Set(srtype, devsNew)
}
