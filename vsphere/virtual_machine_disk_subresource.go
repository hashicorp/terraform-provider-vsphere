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
	r.newData[key] = value
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

// saveID saves the resource ID to internal_id.
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
		strconv.Itoa(int(*disk.UnitNumber)),
	}
	r.set("internal_id", strings.Join(parts, ":"))
}

// expandDiskSettings sets appropriate fields on an existing disk - this is
// used during Create and Update to set attributes to those found in
// configuration.
func (r *resourceVSphereVirtualMachineDisk) expandDiskSettings(disk *types.VirtualDisk) error {
	// Backing settings
	b := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo)
	b.Sharing = r.getWithRestart("disk_mode").(string)
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

// Create creates a vsphere_virtual_machine disk sub-resource.
func (r *resourceVSphereVirtualMachineDisk) Create(l object.VirtualDeviceList) error {
	// Depending on if the controller is already present or not, we need to do one or two things:
	// * Create the controller if it does not exist already
	// * Get an available slot on the newly created or currently present controller
	// This is handled by pickOrCreateController, but we need to check what kind
	// of controller we are working with first.
	var ctlr types.BaseVirtualController
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
		return err
	}

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

	if err := r.expandDiskSettings(disk); err != nil {
		return err
	}

	// Good to go. Set the ID, add the controllers to the device list, and we are done.
	r.saveID(disk, ctlr)

	switch ct := ctlr.(type) {
	case *types.VirtualIDEController:
		l = append(l, ct)
	case *types.ParaVirtualSCSIController:
		l = append(l, ct)
	case *types.VirtualLsiLogicSASController:
		l = append(l, ct)
	default:
		panic(fmt.Errorf("unhandled type %T", ctlr))
	}
	l = append(l, disk)

	return nil
}

// Read reads a vsphere_virtual_machine disk sub-resource and commits the data to the newData layer.
func (r *resourceVSphereVirtualMachineDisk) Read(l object.VirtualDeviceList) error {
	return nil
}

// Update updates a vsphere_virtual_machine disk sub-resource.
func (r *resourceVSphereVirtualMachineDisk) Update(l object.VirtualDeviceList) error {
	return nil
}

// Delete deletes a vsphere_virtual_machine disk sub-resource.
func (r *resourceVSphereVirtualMachineDisk) Delete(l object.VirtualDeviceList) error {
	return nil
}
