package virtualdevice

import (
	"fmt"
	"math"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/datastore"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

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

// DiskSubresourceSchema represents the schema for the disk sub-resource.
func DiskSubresourceSchema() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
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
		"unit_number": {
			Type:         schema.TypeInt,
			Required:     true,
			Description:  "The unique device number for this disk. This number determines where on the SCSI bus this device will be attached.",
			ValidateFunc: validation.IntBetween(0, 29),
		},
		"keep_on_remove": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Set to true to keep the underlying VMDK file when removing this virtual disk from configuration.",
		},
	}
	structure.MergeSchema(s, subresourceSchema())
	return s
}

// DiskSubresource represents a vsphere_virtual_machine disk sub-resource, with
// a complex device lifecycle.
type DiskSubresource struct {
	*Subresource

	// The set hash for the device as it exists when NewDiskSubresource is
	// called.
	ID int
}

// NewDiskSubresource returns a subresource populated with all of the necessary
// fields.
func NewDiskSubresource(client *govmomi.Client, rd *schema.ResourceData, d, old map[string]interface{}) *DiskSubresource {
	sr := &DiskSubresource{
		Subresource: &Subresource{
			schema:       DiskSubresourceSchema(),
			client:       client,
			srtype:       subresourceTypeDisk,
			data:         d,
			olddata:      old,
			resourceData: rd,
		},
	}
	sr.Index = sr.Hash()
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
	o, n := d.GetChange(subresourceTypeDisk)
	// Make an intersection set. Any device in the intersection is a device
	// that is not changing, so we ignore those.
	ids := o.(*schema.Set).Intersection(n.(*schema.Set))
	ods := o.(*schema.Set).Difference(ids)
	nds := n.(*schema.Set).Difference(ids)

	var spec []types.BaseVirtualDeviceConfigSpec

	// Our old and new sets now have an accurate description of devices that may
	// have been added, removed, or changed. Look for removed devices first.
nextOld:
	for _, oe := range ods.List() {
		om := oe.(map[string]interface{})
		for _, ne := range nds.List() {
			nm := ne.(map[string]interface{})
			if om["key"] == nm["key"] {
				continue nextOld
			}
		}
		r := NewDiskSubresource(c, d, om, nil)
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
	for _, ne := range nds.List() {
		nm := ne.(map[string]interface{})
		for _, oe := range ods.List() {
			om := oe.(map[string]interface{})
			if nm["key"] == om["key"] {
				// This is an update
				r := NewDiskSubresource(c, d, nm, om)
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
		r := NewDiskSubresource(c, d, nm, nil)
		cspec, err := r.Create(l)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		l = applyDeviceChange(l, cspec)
		spec = append(spec, cspec...)
		updates = append(updates, r.Data())
	}

	// We are now done! Return the updated device list and config spec. Save updates as well.
	if err := d.Set(subresourceTypeDisk, updates); err != nil {
		return nil, nil, err
	}
	return l, spec, nil
}

// DiskRefreshOperation processes a refresh operation for all of the disks in
// the resource.
//
// This functions similar to DiskApplyOperation, but nothing to change is
// returned, all necessary values are just set and committed to state.
func DiskRefreshOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) error {
	devices := l.Select(func(device types.BaseVirtualDevice) bool {
		if _, ok := device.(*types.VirtualDisk); ok {
			return true
		}
		return false
	})
	curSet := d.Get(subresourceTypeDisk).(*schema.Set).List()
	var newSet []interface{}
	// First check for negative keys. These are freshly added devices that are
	// usually coming into read post-create.
	//
	// If we find what we are looking for, we remove the device from the working
	// set so that we don't try and process it in the next few passes.
	for _, item := range curSet {
		m := item.(map[string]interface{})
		if m["key"].(int) < 1 {
			r := NewDiskSubresource(c, d, m, nil)
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
		for _, item := range curSet {
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
			r := NewDiskSubresource(c, d, m, nil)
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
	for _, device := range devices {
		m := make(map[string]interface{})
		vd := device.GetVirtualDevice()
		ctlr := l.FindByKey(vd.ControllerKey)
		if ctlr == nil {
			return fmt.Errorf("could not find controller with key %d", vd.Key)
		}
		m["key"] = int(vd.Key)
		m["device_address"] = computeDevAddr(vd, ctlr.(types.BaseVirtualController))
		// We want to set keep_on_remove for these disks as well so that they are
		// not destroyed when we remove them in the next TF run.
		m["keep_on_remove"] = true
		r := NewDiskSubresource(c, d, m, nil)
		if err := r.Read(l); err != nil {
			return fmt.Errorf("%s: %s", r.Addr(), err)
		}
		newSet = append(newSet, r.Data())
	}

	return d.Set(subresourceTypeDisk, newSet)
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
		r := NewDiskSubresource(c, d, m, nil)
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

	disk, err := r.createDisk(l)
	if err != nil {
		return nil, fmt.Errorf("error creating disk: %s", err)
	}
	// We now have the controller on which we can create our device on.
	// Assign the disk to a controller.
	ctlr, err := r.assignDisk(l, disk)
	if err != nil {
		return nil, fmt.Errorf("cannot assign disk: %s", err)
	}

	if err := r.expandDiskSettings(disk); err != nil {
		return nil, err
	}

	// Done here. Save ID, push the device to the new device list and return.
	r.SaveDevIDs(disk, ctlr)
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
		return fmt.Errorf("device at %q is not a virtual disk", l.Name(device))
	}
	unit, ctlr, err := r.findControllerInfo(l, disk)
	if err != nil {
		return err
	}
	r.Set("unit_number", unit)
	r.SaveDevIDs(disk, ctlr)

	// Save disk backing settings
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
	return nil
}

// Update updates a vsphere_virtual_machine disk sub-resource.
func (r *DiskSubresource) Update(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	device, err := r.FindVirtualDevice(l)
	if err != nil {
		return nil, fmt.Errorf("cannot find disk device: %s", err)
	}
	disk, ok := device.(*types.VirtualDisk)
	if !ok {
		return nil, fmt.Errorf("device at %q is not a virtual disk", l.Name(device))
	}

	// Has the unit number changed?
	if r.HasChange("unit_number") {
		ctlr, err := r.assignDisk(l, disk)
		if err != nil {
			return nil, fmt.Errorf("cannot assign disk: %s", err)
		}
		r.SaveDevIDs(disk, ctlr)
	}

	// We can now expand the rest of the settings.
	if err := r.expandDiskSettings(disk); err != nil {
		return nil, err
	}

	dspec, err := object.VirtualDeviceList{disk}.ConfigSpec(types.VirtualDeviceConfigSpecOperationEdit)
	if err != nil {
		return nil, err
	}
	return dspec, nil
}

// Delete deletes a vsphere_virtual_machine disk sub-resource.
func (r *DiskSubresource) Delete(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	device, err := r.FindVirtualDevice(l)
	if err != nil {
		return nil, fmt.Errorf("cannot find disk device: %s", err)
	}
	disk, ok := device.(*types.VirtualDisk)
	if !ok {
		return nil, fmt.Errorf("device at %q is not a virtual disk", l.Name(device))
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

// createDisk performs all of the logic for a base virtual disk creation.
func (r *DiskSubresource) createDisk(l object.VirtualDeviceList) (*types.VirtualDisk, error) {
	dsID := r.Get("datastore_id").(string)
	if dsID == "" {
		// Default to the default datastore
		dsID = r.resourceData.Get("datastore_id").(string)
	}
	ds, err := datastore.FromID(r.client, dsID)
	if err != nil {
		return nil, err
	}
	dsref := ds.Reference()

	disk := &types.VirtualDisk{
		VirtualDevice: types.VirtualDevice{
			Backing: &types.VirtualDiskFlatVer2BackingInfo{
				VirtualDeviceFileBackingInfo: types.VirtualDeviceFileBackingInfo{
					FileName:  ds.Path(r.Get("path").(string)),
					Datastore: &dsref,
				},
			},
		},
	}
	// Set a new device key for this device
	disk.Key = l.NewKey()
	return disk, nil
}

// assignDisk takes a unit number and assigns it correctly to a controller on
// the SCSI bus. An error is returned if the assigned unit number is taken.
func (r *DiskSubresource) assignDisk(l object.VirtualDeviceList, disk *types.VirtualDisk) (types.BaseVirtualController, error) {
	number := r.Get("unit_number").(int)
	// Figure out the bus number, and look up the SCSI controller that matches
	// that. You can attach 15 disks to a SCSI controller, and we allow a maximum
	// of 30 devices.
	bus := number / 15
	// Also determine the unit number on that controller.
	unit := int32(math.Mod(float64(number), 15))

	// Find the controller.
	ctlr, err := r.ControllerForCreateUpdate(l, SubresourceControllerTypeSCSI, bus)
	if err != nil {
		return nil, err
	}

	// Build the unit list.
	units := make([]bool, 16)
	// Reserve the SCSI unit number
	scsiUnit := ctlr.(types.BaseVirtualSCSIController).GetVirtualSCSIController().ScsiCtlrUnitNumber
	units[scsiUnit] = true

	ckey := ctlr.GetVirtualController().Key

	for _, device := range l {
		d := device.GetVirtualDevice()
		if d.ControllerKey != ckey || d.UnitNumber == nil {
			continue
		}
		units[*d.UnitNumber] = true
	}

	// We now have a valid list of units. If we need to, shift up the desired
	// unit number so it's not taking the unit of the controller itself.
	if unit >= scsiUnit {
		unit++
	}

	if units[unit] {
		return nil, fmt.Errorf("unit number %d on SCSI bus %d is in use", unit, bus)
	}

	// If we made it this far, we are good to go!
	disk.ControllerKey = ctlr.GetVirtualController().Key
	disk.UnitNumber = &unit
	return ctlr, nil
}

// findControllerInfo determines the normalized unit number for the disk device
// based on the SCSI controller and unit number it's connected to. The
// controller is also returned.
func (r *Subresource) findControllerInfo(l object.VirtualDeviceList, disk *types.VirtualDisk) (int, types.BaseVirtualController, error) {
	ctlr := l.FindByKey(disk.ControllerKey)
	if ctlr == nil {
		return -1, nil, fmt.Errorf("could not find disk controller with key %d for disk key %d", disk.ControllerKey, disk.Key)
	}
	if disk.UnitNumber == nil {
		return -1, nil, fmt.Errorf("unit number on disk key %d is unset", disk.Key)
	}
	unit := *disk.UnitNumber
	if unit > ctlr.(types.BaseVirtualSCSIController).GetVirtualSCSIController().ScsiCtlrUnitNumber {
		unit--
	}
	return int(unit), ctlr.(types.BaseVirtualController), nil
}
