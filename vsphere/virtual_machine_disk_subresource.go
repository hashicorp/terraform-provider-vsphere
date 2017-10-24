package vsphere

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

const (
	resourceVSphereVirtualMachineDiskControllerTypeIDE  = "ide"
	resourceVSphereVirtualMachineDiskControllerTypeSCSI = "scsi"

	resourceVSphereVirtualMachineDiskSCSITypeParaVirtual = "pvscsi"
	resourceVSphereVirtualMachineDiskSCSITypeLsiLogicSAS = "lsilogic-sas"
)

var resourceVSphereVirtualMachineDiskControllerTypeAllowedValues = []string{
	resourceVSphereVirtualMachineDiskControllerTypeIDE,
	resourceVSphereVirtualMachineDiskControllerTypeSCSI,
}

var resourceVSphereVirtualMachineDiskModeAllowedValues = []string{
	string(types.VirtualDiskModePersistent),
	string(types.VirtualDiskModeNonpersistent),
	string(types.VirtualDiskModeUndoable),
	string(types.VirtualDiskModeIndependent_persistent),
	string(types.VirtualDiskModeIndependent_nonpersistent),
	string(types.VirtualDiskModeAppend),
}

var resourceVSphereVirtualMachineDiskSharingAllowedValues = []string{
	string(types.VirtualDiskSharingSharingNone),
	string(types.VirtualDiskSharingSharingMultiWriter),
}

var resourceVSphereVirtualMachineDiskSCSITypeAllowedValues = []string{
	resourceVSphereVirtualMachineDiskSCSITypeParaVirtual,
	resourceVSphereVirtualMachineDiskSCSITypeLsiLogicSAS,
}

// resourceVSphereVirtualMachineDisk defines a vsphere_virtual_machine disk
// sub-resource.
//
// The workflow here is CRUD-like, and designed to be portable to other uses in
// the future, however various changes are made to the interface to account for
// the fact that this is not necessarily a fully-fledged resource in its own
// right.
type resourceVSphereVirtualMachineDisk struct {
	// The old resource data.
	oldData map[string]interface{}

	// The new resource data.
	newData map[string]interface{}

	// This is flagged if anything in the CRUD process required a VM restart. The
	// parent CRUD is responsible for flagging the appropriate information and
	// doing the necessary restart before applying the resulting ConfigSpec.
	restart bool
}

// resourceVSphereVirtualMachineDiskSchema returns the schema for the disk
// sub-resource.
func resourceVSphereVirtualMachineDiskSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		// VirtualDiskFlatVer2BackingInfo
		"datastore_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "The datastore ID for this virtual disk, if different than the virtual machine.",
		},
		"path": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "An optional path for the disk. If this disk exists already, the disk is attached rather than created. Any folders in the path need to exist when disk is added.",
		},
		"disk_mode": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      string(types.VirtualDiskModePersistent),
			Description:  "The mode of this this virtual disk for purposes of writes and snapshotting. Can be one of append, independent_nonpersistent, independent_persistent, nonpersistent, persistent, or undoable.",
			ValidateFunc: validation.StringInSlice(resourceVSphereVirtualMachineDiskModeAllowedValues, false),
		},
		"eagerly_scrub": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "The virtual disk file zeroing policy when thin_provision is not true. The default is false, which lazily-zeros the disk, speeding up thick-provisioned disk creation time.",
		},
		"disk_sharing": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      string(types.VirtualDiskSharingSharingNone),
			Description:  "The sharing mode of this virtual disk. Can be one of sharingMultiWriter or sharingNone.",
			ValidateFunc: validation.StringInSlice(resourceVSphereVirtualMachineDiskSharingAllowedValues, false),
		},
		"thin_provisioned": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "If true, this disk is thin provisioned, with space for the file being allocated on an as-needed basis.",
		},
		"write_through": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "If true, writes for this disk are sent directly to the filesystem immediately instead of being buffered.",
		},

		// StorageIOAllocationInfo
		"io_limit": {
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      -1,
			Description:  "The upper limit of IOPS that this disk can use.",
			ValidateFunc: validation.IntAtLeast(-1),
		},
		"io_reservation": {
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      0,
			Description:  "The I/O guarantee that this disk has, in IOPS.",
			ValidateFunc: validation.IntAtLeast(0),
		},
		"io_share_level": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      string(types.SharesLevelNormal),
			Description:  "The share allocation level for this disk. Can be one of low, normal, high, or custom.",
			ValidateFunc: validation.StringInSlice(sharesLevelAllowedValues, false),
		},
		"io_share_count": {
			Type:         schema.TypeInt,
			Optional:     true,
			Computed:     true,
			Description:  "The share count for this disk when the share level is custom.",
			ValidateFunc: validation.IntAtLeast(0),
		},

		// VirtualDisk/Other complex stuff
		"size": {
			Type:         schema.TypeInt,
			Required:     true,
			Description:  "The size of the disk, in GB.",
			ValidateFunc: validation.IntAtLeast(1),
		},
		"controller_type": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      resourceVSphereVirtualMachineDiskControllerTypeSCSI,
			Description:  "The controller type. Can be one of ide or scsi.",
			ValidateFunc: validation.StringInSlice(resourceVSphereVirtualMachineDiskControllerTypeAllowedValues, false),
		},
		"scsi_controller_type": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      resourceVSphereVirtualMachineDiskSCSITypeLsiLogicSAS,
			Description:  "The SCSI controller type. Can be one of pvscsi or lsilogic-sas.",
			ValidateFunc: validation.StringInSlice(resourceVSphereVirtualMachineDiskSCSITypeAllowedValues, false),
		},
		"keep_on_remove": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Set to true to keep the underlying VMDK file when removing this virtual disk from configuration.",
		},
		"controller_bus_number": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The bus index for the controller that this device is attached to.",
		},
		"controller_key": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The unique device ID for the controller this device is attached to.",
		},
		"unit_number": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The unit number of the device on the controller.",
		},
		"key": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The unique device ID for this device within the virtual machine configuration.",
		},
		"internal_id": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The internally-computed ID of this resource, local to Terraform - this is controller_type:controller_bus_number:unit_number.",
		},
	}
}

// pickOrCreateController either finds a device of a specific time with an
// available slot, or creates a new one. An error is returned if there is a
// problem anywhere in this process or not possible.
//
// Note that this does not push the controller to the device list - this is
// done outside of this function, to keep things atomic at the end.
func pickOrCreateController(l object.VirtualDeviceList, kind types.BaseVirtualController) (types.BaseVirtualController, error) {
	ctlr := l.PickController(kind)
	if ctlr == nil {
		var nc types.BaseVirtualDevice
		var err error
		switch kind.(type) {
		case *types.VirtualIDEController:
			nc, err = l.CreateIDEController()
			ctlr = nc.(*types.VirtualIDEController)
		case *types.ParaVirtualSCSIController:
			nc, err = l.CreateSCSIController(resourceVSphereVirtualMachineDiskSCSITypeParaVirtual)
			ctlr = nc.(*types.ParaVirtualSCSIController)
		case *types.VirtualLsiLogicSASController:
			nc, err = l.CreateSCSIController(resourceVSphereVirtualMachineDiskSCSITypeLsiLogicSAS)
			ctlr = nc.(*types.VirtualLsiLogicSASController)
		default:
			return nil, fmt.Errorf("invalid controller type: %T", kind)
		}
		if err != nil {
			return nil, err
		}
	}
	return ctlr, nil
}

// get gets the field from the new set of resource data.
func (r *resourceVSphereVirtualMachineDisk) get(key string) interface{} {
	return r.newData[key]
}

// set sets the field in the newData set.
func (r *resourceVSphereVirtualMachineDisk) set(key string, value interface{}) {
	r.newData[key] = deRef(value)
}

// hasChange checks to see if a field has been modified and returns true if it
// has.
func (r *resourceVSphereVirtualMachineDisk) hasChange(key string) bool {
	return r.oldData[key] != r.newData[key]
}

// getChange returns the old and new values for the supplied key.
func (r *resourceVSphereVirtualMachineDisk) getChange(key string) (interface{}, interface{}) {
	return r.oldData[key], r.newData[key]
}

// getWithRestart checks to see if a field has been modified, returns the new
// value, and sets restart if it has changed.
func (r *resourceVSphereVirtualMachineDisk) getWithRestart(key string) interface{} {
	if r.hasChange(key) {
		r.restart = true
	}
	return r.get(key)
}

// getWithVeto returns the value specified by key, but returns an error if it
// has changed. The intention here is to block changes to the resource in a
// fashion that would otherwise result in forcing a new resource.
func (r *resourceVSphereVirtualMachineDisk) getWithVeto(key string) (interface{}, error) {
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
func (r *resourceVSphereVirtualMachineDisk) saveID(disk *types.VirtualDisk, ctlr types.BaseVirtualController) {
	vc := ctlr.GetVirtualController()
	parts := []string{
		r.get("controller_type").(string),
		strconv.Itoa(int(vc.BusNumber)),
		strconv.Itoa(int(deRef(disk.UnitNumber).(int32))),
	}
	r.set("controller_bus_number", vc.BusNumber)
	r.set("controller_key", vc.Key)
	r.set("unit_number", disk.UnitNumber)
	r.set("internal_id", strings.Join(parts, ":"))
}

// id returns the internal_id attribute in the subresource. This function
// exists mainly as a functional counterpart to saveID.
func (r *resourceVSphereVirtualMachineDisk) id() string {
	return r.get("internal_id").(string)
}

// splitVirtualMachineDiskID splits an ID into its inparticular parts and
// asserts that we have all the correct data.
func splitVirtualMachineDiskID(id string) (string, int, int, error) {
	parts := strings.Split(id, ":")
	if len(parts) < 3 {
		return "", 0, 0, fmt.Errorf("invalid controller type %q found in ID", id)
	}
	ct, cbs, dus := parts[0], parts[1], parts[2]
	cb, cbe := strconv.Atoi(cbs)
	du, due := strconv.Atoi(dus)
	var found bool
	for _, v := range resourceVSphereVirtualMachineDiskControllerTypeAllowedValues {
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

// findVirtualMachineDiskInListControllerSelectFunc returns a function that can
// be used with VirtualDeviceList.Select to locate a controller device based on
// the criteria that we have laid out.
func findVirtualMachineDiskInListControllerSelectFunc(ct string, cb int) func(types.BaseVirtualDevice) bool {
	return func(device types.BaseVirtualDevice) bool {
		var ctlr types.BaseVirtualController
		switch ct {
		case resourceVSphereVirtualMachineDiskControllerTypeIDE:
			if v, ok := device.(*types.VirtualIDEController); ok {
				ctlr = v
				goto controllerFound
			}
			return false
		case resourceVSphereVirtualMachineDiskControllerTypeSCSI:
			switch v := device.(type) {
			case *types.ParaVirtualSCSIController:
				ctlr = v
				goto controllerFound
			case *types.VirtualLsiLogicSASController:
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

// findVirtualMachineDiskInListDiskSelectFunc returns a function that can be
// used with VirtualDeviceList.Select to locate a disk device based on its
// controller device key, and the disk number on the device.
func findVirtualMachineDiskInListDiskSelectFunc(ckey int32, du int) func(types.BaseVirtualDevice) bool {
	return func(device types.BaseVirtualDevice) bool {
		disk, ok := device.(*types.VirtualDisk)
		if !ok {
			return false
		}
		if disk.ControllerKey == ckey && disk.UnitNumber != nil && *disk.UnitNumber == int32(du) {
			return true
		}
		return false
	}
}

// findVirtualMachineDiskInList looks for a specific device in the device list given a
// specific disk device key. nil is returned if no device is found.
func findVirtualMachineDiskInList(l object.VirtualDeviceList, id string) (*types.VirtualDisk, error) {
	ct, cb, du, err := splitVirtualMachineDiskID(id)
	if err != nil {
		return nil, err
	}

	// find the controller
	csf := findVirtualMachineDiskInListControllerSelectFunc(ct, cb)
	ctlrs := l.Select(csf)
	if len(ctlrs) != 1 {
		return nil, fmt.Errorf("invalid controller result - %d results returned (expected 1): type %q, bus number: %d", len(ctlrs), ct, cb)
	}
	ctlr := ctlrs[0]

	// find the disk
	ckey := ctlr.GetVirtualDevice().Key
	dsf := findVirtualMachineDiskInListDiskSelectFunc(ckey, du)
	disks := l.Select(dsf)
	if len(disks) != 1 {
		return nil, fmt.Errorf("invalid disk result - %d results returned (expected 1): controller key %q, disk number: %d", len(disks), ckey, du)
	}
	disk := disks[0]
	return disk.(*types.VirtualDisk), nil
}

// expandDiskSettings sets appropriate fields on an existing disk - this is
// used during Create and Update to set attributes to those found in
// configuration.
func (r *resourceVSphereVirtualMachineDisk) expandDiskSettings(disk *types.VirtualDisk) error {
	// Backing settings
	b := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo)
	b.DiskMode = r.getWithRestart("disk_mode").(string)
	b.WriteThrough = boolPtr(r.getWithRestart("write_through").(bool))
	b.Sharing = r.getWithRestart("disk_sharing").(string)

	var err error
	var v interface{}
	if v, err = r.getWithVeto("thin_provisioned"); err != nil {
		return err
	}
	b.ThinProvisioned = boolPtr(v.(bool))

	if v, err = r.getWithVeto("eagerly_scrub"); err != nil {
		return err
	}
	b.EagerlyScrub = boolPtr(v.(bool))

	if v, err = r.getWithVeto("path"); err != nil {
		return err
	}
	b.FileName = v.(string)

	// Disk settings
	os, ns := r.getChange("size")
	if os.(int) > ns.(int) {
		return fmt.Errorf("virtual disks cannot be shrunk")
	}
	disk.CapacityInBytes = gbToByte(ns.(int))

	alloc := &types.StorageIOAllocationInfo{
		Limit:       int64Ptr(int64(r.get("io_limit").(int))),
		Reservation: int32Ptr(int32(r.get("io_reservation").(int))),
		Shares: &types.SharesInfo{
			Shares: int32(r.get("io_share_count").(int)),
			Level:  types.SharesLevel(r.get("io_share_level").(string)),
		},
	}
	disk.StorageIOAllocation = alloc

	return nil
}

// flattenDiskSettings sets appropriate attributes on a disk resource from the
// passed in VirtualDisk.
//
// Some computed attributes that generally have to do with device workflow are
// not set here, and are up to the caller to set.
func (r *resourceVSphereVirtualMachineDisk) flattenDiskSettings(disk *types.VirtualDisk) error {
	// Backing settings
	b := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo)
	r.set("disk_mode", b.DiskMode)
	r.set("write_through", b.WriteThrough)
	r.set("sharing", b.Sharing)
	r.set("thin_provisioned", b.ThinProvisioned)
	r.set("eagerly_scrub", b.EagerlyScrub)
	r.set("path", b.FileName)

	// Disk settings
	r.set("size", byteToGB(disk.CapacityInBytes))

	if disk.StorageIOAllocation != nil {
		r.set("io_limit", disk.StorageIOAllocation.Limit)
		r.set("io_reservation", disk.StorageIOAllocation.Reservation)
		if disk.StorageIOAllocation.Shares != nil {
			r.set("io_share_count", disk.StorageIOAllocation.Shares.Shares)
			r.set("io_share_level", disk.StorageIOAllocation.Shares.Level)
		}
	}

	// Device key
	r.set("key", disk.Key)
	return nil
}

// controllerForCreateUpdate wraps the controller selection logic mainly used
// for creation so that we can re-use it in Update.
//
// If the controller is new, it's returned as the second return value, as a
// VirtualDeviceList, for easy appending to outbound devices and the working
// set.
func (r *resourceVSphereVirtualMachineDisk) controllerForCreateUpdate(l object.VirtualDeviceList) (types.BaseVirtualController, object.VirtualDeviceList, error) {
	var ctlr types.BaseVirtualController
	var newDevices object.VirtualDeviceList
	var err error
	ct := r.get("controller_type").(string)
	sct := r.get("scsi_controller_type").(string)
	switch ct {
	case resourceVSphereVirtualMachineDiskControllerTypeIDE:
		ctlr, err = pickOrCreateController(l, &types.VirtualIDEController{})
	case resourceVSphereVirtualMachineDiskControllerTypeSCSI:
		switch sct {
		case resourceVSphereVirtualMachineDiskSCSITypeParaVirtual:
			ctlr, err = pickOrCreateController(l, &types.ParaVirtualSCSIController{})
		case resourceVSphereVirtualMachineDiskSCSITypeLsiLogicSAS:
			ctlr, err = pickOrCreateController(l, &types.VirtualLsiLogicSASController{})
		}
	}
	if err != nil {
		return nil, nil, err
	}

	// Is this a new controller? If so, we need to push this to our working
	// device set so that its device key is accounted for, in addition to the
	// list of new devices that we are returning as part of the device creation,
	// so that they can be added to the ConfigSpec properly.
	if ctlr.GetVirtualController().Key < 0 {
		switch ct := ctlr.(type) {
		case *types.VirtualIDEController:
			newDevices = append(newDevices, ct)
		case *types.ParaVirtualSCSIController:
			newDevices = append(newDevices, ct)
		case *types.VirtualLsiLogicSASController:
			newDevices = append(newDevices, ct)
		default:
			panic(fmt.Errorf("unhandled controller type %T", ctlr))
		}
	}
	return ctlr, newDevices, nil
}

// Create creates a vsphere_virtual_machine disk sub-resource.
func (r *resourceVSphereVirtualMachineDisk) Create(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	// Depending on if the controller is already present or not, we need to do
	// one or two things:
	// * Create the controller if it does not exist already
	// * Get an available slot on the newly created or currently present
	// controller This is handled by pickOrCreateController, but we need to check
	// what kind of controller we are working with first.
	var newDevices object.VirtualDeviceList
	var ctlr types.BaseVirtualController
	ctlr, ncl, err := r.controllerForCreateUpdate(l)
	if err != nil {
		return nil, err
	}
	l = append(l, ncl...)
	newDevices = append(newDevices, ncl...)

	// We now have the controller on which we can create our device on.
	var dsRef types.ManagedObjectReference
	dsID := r.get("datastore_id").(string)
	path := r.get("path").(string)
	if dsID != "" {
		dsRef.Type = "Datastore"
		dsRef.Value = dsID
	}
	disk := l.CreateDisk(ctlr, dsRef, path)
	if dsID == "" {
		// CreateDisk does not allow you to pass nil as a datastore reference
		// currently, so we have to nil out the value after the fact.
		disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo).Datastore = nil
	}
	// Set a new device key for this device as CreateDisk does not do it for us
	// right now.
	disk.Key = l.NewKey()

	if err := r.expandDiskSettings(disk); err != nil {
		return nil, err
	}

	// Done here. Save ID, push the device to the new device list and return.
	r.saveID(disk, ctlr)
	newDevices = append(newDevices, disk)
	return newDevices.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
}

// Read reads a vsphere_virtual_machine disk sub-resource and commits the data
// to the newData layer.
func (r *resourceVSphereVirtualMachineDisk) Read(l object.VirtualDeviceList) error {
	id := r.id()
	disk, err := findVirtualMachineDiskInList(l, id)
	if err != nil {
		return fmt.Errorf("cannot find disk device: %s", err)
	}
	return r.flattenDiskSettings(disk)
}

// Update updates a vsphere_virtual_machine disk sub-resource.
func (r *resourceVSphereVirtualMachineDisk) Update(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	id := r.id()
	disk, err := findVirtualMachineDiskInList(l, id)
	if err != nil {
		return nil, fmt.Errorf("cannot find disk device: %s", err)
	}

	var updateList object.VirtualDeviceList

	// There's 2 main update operations:
	//
	// * Controller modification: A modification where we are changing the
	// controller type (ie: IDE to SCSI, different SCSI device type)
	// * Everything else, really.
	//
	// The former requires us to essentially detach this disk from one controller
	// and attach it to another. This is still just an edit operation, but it's
	// still a little more complex than just expanding the options into the disk
	// device.
	if r.hasChange("controller_type") || r.hasChange("scsi_controller_type") {
		var ctlr types.BaseVirtualController
		ctlr, ncl, err := r.controllerForCreateUpdate(l)
		if err != nil {
			return nil, err
		}
		l = append(l, ncl...)
		updateList = append(updateList, ncl...)
		// This operation also requires a restart, so flag that now.
		r.restart = true
		// Finally, our device needs a new ID (not key, but the internal ID we use
		// to track things in lieu of keys). This ultimately means that the new
		// resource data that comes out of this function should be either be set
		// post-update operation (in the parent resource), or set with partial mode
		// on, which should be turned off when the update operation is successful
		// (probably pretty much right after ReconfigureVM_Task).
		r.saveID(disk, ctlr)
	}

	// We can now expand the rest of the settings.
	if err := r.expandDiskSettings(disk); err != nil {
		return nil, err
	}

	updateList = append(updateList, disk)
	return updateList.ConfigSpec(types.VirtualDeviceConfigSpecOperationEdit)
}

// Delete deletes a vsphere_virtual_machine disk sub-resource.
func (r *resourceVSphereVirtualMachineDisk) Delete(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	id := r.id()
	disk, err := findVirtualMachineDiskInList(l, id)
	if err != nil {
		return nil, fmt.Errorf("cannot find disk device: %s", err)
	}
	deleteSpec, err := object.VirtualDeviceList{disk}.ConfigSpec(types.VirtualDeviceConfigSpecOperationRemove)
	if err != nil {
		return nil, err
	}
	if r.get("keep_on_remove").(bool) {
		// Clear file operation so that the disk is kept on remove.
		deleteSpec[0].GetVirtualDeviceConfigSpec().FileOperation = ""
	}
	return deleteSpec, nil
}
