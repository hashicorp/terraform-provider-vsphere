package virtualdevice

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/datastore"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

var diskSubresourceControllerTypeAllowedValues = []string{
	SubresourceControllerTypeIDE,
	SubresourceControllerTypeParaVirtual,
	SubresourceControllerTypeLsiLogicSAS,
}

var diskSubresourceModeAllowedValues = []string{
	string(types.VirtualDiskModePersistent),
	string(types.VirtualDiskModeNonpersistent),
	string(types.VirtualDiskModeUndoable),
	string(types.VirtualDiskModeIndependent_persistent),
	string(types.VirtualDiskModeIndependent_nonpersistent),
	string(types.VirtualDiskModeAppend),
}

var diskSubresourceSharingAllowedValues = []string{
	string(types.VirtualDiskSharingSharingNone),
	string(types.VirtualDiskSharingSharingMultiWriter),
}

// diskSubresourceSchema represents the schema for the disk sub-resource.
func diskSubresourceSchema() map[string]*schema.Schema {
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
			ValidateFunc: validation.StringInSlice(diskSubresourceModeAllowedValues, false),
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
			ValidateFunc: validation.StringInSlice(diskSubresourceSharingAllowedValues, false),
		},
		"thin_provisioned": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
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
			Default:      SubresourceControllerTypeLsiLogicSAS,
			Description:  "The controller type. Can be one of ide, pvscsi, or lsilogic-sas.",
			ValidateFunc: validation.StringInSlice(diskSubresourceControllerTypeAllowedValues, false),
		},
		"keep_on_remove": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Set to true to keep the underlying VMDK file when removing this virtual disk from configuration.",
		},
		"key": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The unique device ID for this device within the virtual machine configuration.",
		},
	}
}

// DiskSubresource represents a vsphere_virtual_machine disk sub-resource, with
// a complex device lifecycle.
type DiskSubresource struct {
	*Subresource
}

// NewDiskSubresource returns a subresource populated with all of the necessary
// fields.
func NewDiskSubresource(client *govmomi.Client, index int, d *schema.ResourceData) SubresourceInstance {
	sr := &DiskSubresource{
		Subresource: &Subresource{
			schema: diskSubresourceSchema(),
			client: client,
			srtype: subresourceTypeDisk,
			index:  index,
			data:   d,
		},
	}
	return sr
}

// DiskApplyOperation processes an apply operation for all disks in the
// resource.
//
// The function takes the root resource's ResourceData, the provider
// connection, and the device list as known to vSphere at the start of this
// operation. All disk operations are carried out, with both the complete,
// updated, VirtualDeviceList, and the complete list of changes returned as a
// slice of BaseVirtualDeviceConfigSpec.
func DiskApplyOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) (object.VirtualDeviceList, []types.BaseVirtualDeviceConfigSpec, error) {
	return deviceApplyOperation(d, c, l, subresourceTypeDisk, NewDiskSubresource)
}

// DiskRefreshOperation processes a refresh operation for all of the disks in
// the resource.
//
// This functions similar to DiskApplyOperation, but nothing to change is
// returned, all necessary values are just set and committed to state.
func DiskRefreshOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) error {
	return deviceRefreshOperation(d, c, l, subresourceTypeDisk, NewDiskSubresource)
}

// DiskDestroyOperation process the destroy operation for virtual disks.
//
// Disks are the only real operation that require special destroy logic, and
// that's because we want to check to make sure that we detach any disks that
// need to be simply detached (not deleted) before we destroy the entire
// virtual machine, as that would take those disks with it.
func DiskDestroyOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	// All we are doing here is getting a config spec for detaching the disks
	// that we need to detach, so we don't need the vast majority of the stateful
	// logic that is in deviceApplyOperation.
	ds := d.Get(subresourceTypeDisk).(*schema.Set)

	var spec []types.BaseVirtualDeviceConfigSpec

	for _, oe := range ds.List() {
		m := oe.(map[string]interface{})
		if !m["keep_on_remove"].(bool) {
			// We don't care about disks we haven't set to keep
			continue
		}
		r := NewDiskSubresource(c, m["index"].(int), d)
		dspec, err := r.Delete(l)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		applyDeviceChange(l, dspec)
		spec = append(spec, dspec...)
	}

	return spec, nil
}

// Create creates a vsphere_virtual_machine disk sub-resource.
func (r *DiskSubresource) Create(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	var spec []types.BaseVirtualDeviceConfigSpec
	var ctlr types.BaseVirtualController
	ctlr, cspec, err := r.ControllerForCreateUpdate(l, r.Get("controller_type").(string))
	if err != nil {
		return nil, err
	}
	if len(cspec) > 0 {
		l = append(l, cspec[0].GetVirtualDeviceConfigSpec().Device)
		spec = append(spec, cspec...)
	}

	// We now have the controller on which we can create our device on.
	dsID := r.Get("datastore_id").(string)
	if dsID == "" {
		// Default to the default datastore
		dsID = r.data.Get("datastore_id").(string)
	}
	ds, err := datastore.FromID(r.client, dsID)
	if err != nil {
		return nil, fmt.Errorf("could not locate datastore: %s", err)
	}
	disk := l.CreateDisk(ctlr, ds.Reference(), "")
	// We need to set the backing path manually as CreateDisk currently breaks
	// FileNames with just datastores in them.
	disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo).FileName = ds.Path(r.Get("path").(string))
	// Set a new device key for this device as CreateDisk does not do it for us
	// right now.
	disk.Key = l.NewKey()
	// Ensure the device starts connected
	l.Connect(disk)

	if err := r.expandDiskSettings(disk); err != nil {
		return nil, err
	}

	// Done here. Save ID, push the device to the new device list and return.
	r.SaveID(disk, ctlr)
	dspec, err := object.VirtualDeviceList{disk}.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
	if err != nil {
		return nil, err
	}
	spec = append(spec, dspec...)
	return spec, nil
}

// Read reads a vsphere_virtual_machine disk sub-resource and commits the data
// to the newData layer.
func (r *DiskSubresource) Read(l object.VirtualDeviceList) error {
	device, err := r.FindVirtualDevice(l)
	if err != nil {
		return fmt.Errorf("cannot find disk device: %s", err)
	}
	disk, ok := device.(*types.VirtualDisk)
	if !ok {
		return fmt.Errorf("device at %q is not a virtual disk", r.ID())
	}
	// Is this disk not managed by Terraform? If not, we want to flag
	// keep_on_remove, just to make sure that that we don't blow this disk away
	// when we remove it on the next TF run.
	if r.index >= orpahnedDeviceMinIndex {
		r.Set("keep_on_remove", true)
	}
	return r.flattenDiskSettings(disk)
}

// Update updates a vsphere_virtual_machine disk sub-resource.
func (r *DiskSubresource) Update(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	device, err := r.FindVirtualDevice(l)
	if err != nil {
		return nil, fmt.Errorf("cannot find disk device: %s", err)
	}
	disk, ok := device.(*types.VirtualDisk)
	if !ok {
		return nil, fmt.Errorf("device at %q is not a virtual disk", r.ID())
	}

	// We maintain the final update spec in place, versus just the simple device
	// list, as we are possibly creating controllers here.
	var spec []types.BaseVirtualDeviceConfigSpec

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
	if r.HasChange("controller_type") {
		var ctlr types.BaseVirtualController
		ctlr, cspec, err := r.ControllerForCreateUpdate(l, r.Get("controller_type").(string))
		if err != nil {
			return nil, err
		}
		if len(cspec) > 0 {
			// VirtaulDeviceList.ConfigSpec never returns anything else other than
			// VirtualDeviceConfigSpec at this point, so it's safe to assert here.
			l = append(l, cspec[0].(*types.VirtualDeviceConfigSpec).Device)
			spec = append(spec, cspec...)
		}
		// This operation also requires a restart, so flag that now.
		r.SetRestart()
		// Finally, our device needs a new ID (not key, but the internal ID we use
		// to track things in lieu of keys). This ultimately means that the new
		// resource data that comes out of this function should be either be set
		// post-update operation (in the parent resource), or set with partial mode
		// on, which should be turned off when the update operation is successful
		// (probably pretty much right after ReconfigureVM_Task).
		r.SaveID(disk, ctlr)
	}

	// We can now expand the rest of the settings.
	if err := r.expandDiskSettings(disk); err != nil {
		return nil, err
	}

	dspec, err := object.VirtualDeviceList{disk}.ConfigSpec(types.VirtualDeviceConfigSpecOperationEdit)
	if err != nil {
		return nil, err
	}
	spec = append(spec, dspec...)
	return spec, nil
}

// Delete deletes a vsphere_virtual_machine disk sub-resource.
func (r *DiskSubresource) Delete(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	device, err := r.FindVirtualDevice(l)
	if err != nil {
		return nil, fmt.Errorf("cannot find disk device: %s", err)
	}
	disk, ok := device.(*types.VirtualDisk)
	if !ok {
		return nil, fmt.Errorf("device at %q is not a virtual disk", r.ID())
	}
	deleteSpec, err := object.VirtualDeviceList{disk}.ConfigSpec(types.VirtualDeviceConfigSpecOperationRemove)
	if err != nil {
		return nil, err
	}
	if r.Get("keep_on_remove").(bool) {
		// Clear file operation so that the disk is kept on remove.
		deleteSpec[0].GetVirtualDeviceConfigSpec().FileOperation = ""
	}
	return deleteSpec, nil
}

// expandDiskSettings sets appropriate fields on an existing disk - this is
// used during Create and Update to set attributes to those found in
// configuration.
func (r *DiskSubresource) expandDiskSettings(disk *types.VirtualDisk) error {
	// Backing settings
	b := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo)
	b.DiskMode = r.GetWithRestart("disk_mode").(string)
	b.WriteThrough = structure.BoolPtr(r.GetWithRestart("write_through").(bool))
	b.Sharing = r.GetWithRestart("disk_sharing").(string)

	var err error
	var v interface{}
	if v, err = r.GetWithVeto("thin_provisioned"); err != nil {
		return err
	}
	b.ThinProvisioned = structure.BoolPtr(v.(bool))

	if v, err = r.GetWithVeto("eagerly_scrub"); err != nil {
		return err
	}
	b.EagerlyScrub = structure.BoolPtr(v.(bool))

	// Disk settings
	os, ns := r.GetChange("size")
	if os.(int) > ns.(int) {
		return fmt.Errorf("virtual disks cannot be shrunk")
	}
	disk.CapacityInBytes = structure.GiBToByte(ns.(int))
	disk.CapacityInKB = disk.CapacityInBytes / 1024

	alloc := &types.StorageIOAllocationInfo{
		Limit:       structure.Int64Ptr(int64(r.Get("io_limit").(int))),
		Reservation: structure.Int32Ptr(int32(r.Get("io_reservation").(int))),
		Shares: &types.SharesInfo{
			Shares: int32(r.Get("io_share_count").(int)),
			Level:  types.SharesLevel(r.Get("io_share_level").(string)),
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
func (r *DiskSubresource) flattenDiskSettings(disk *types.VirtualDisk) error {
	// Backing settings
	b := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo)
	r.Set("disk_mode", b.DiskMode)
	r.Set("write_through", b.WriteThrough)
	r.Set("disk_sharing", b.Sharing)
	r.Set("thin_provisioned", b.ThinProvisioned)
	r.Set("eagerly_scrub", b.EagerlyScrub)
	r.Set("datastore_id", b.Datastore.Value)
	// Save path properly
	dp := &object.DatastorePath{}
	if ok := dp.FromString(b.FileName); !ok {
		return fmt.Errorf("could not parse path from filename: %s", b.FileName)
	}
	r.Set("path", dp.Path)

	// Disk settings
	r.Set("size", structure.ByteToGiB(disk.CapacityInBytes))

	if disk.StorageIOAllocation != nil {
		r.Set("io_limit", disk.StorageIOAllocation.Limit)
		r.Set("io_reservation", disk.StorageIOAllocation.Reservation)
		if disk.StorageIOAllocation.Shares != nil {
			r.Set("io_share_count", disk.StorageIOAllocation.Shares.Shares)
			r.Set("io_share_level", disk.StorageIOAllocation.Shares.Level)
		}
	}

	// Device key
	r.Set("key", disk.Key)
	return nil
}
