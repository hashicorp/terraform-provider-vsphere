package virtualdevice

import (
	"errors"
	"fmt"
	"log"
	"math"
	"path"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/mitchellh/copystructure"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/datastore"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/viapi"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

// diskReadNameMismatchError is a "friendly" error message that's displayed if
// for some reason there is a mismatch between the original name of a disk, and
// the name found by Terraform. This can happen if disk device configuration is
// significantly changed manually, or if a snapshot chain is broken (can happen
// when linked clones are storage vMotioned).
const diskReadNameMismatchError = `
!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

Terraform discovered a different virtual disk than expected:

Virtual machine: %s
Device key:      %d
Device address:  %s
Expected disk:   %s
Actual disk:     %s

Possible causes are:

* The disk configuration was significantly modified outside of Terraform.
* The disk has snapshots and Terraform cannot trace back to the original disk.
* The virtual machine is a linked clone that was migrated to another datastore

Terraform cannot continue. If at all possible, manually correct the
configuration of the virtual machine, run "terraform plan", and carefully
review the changes, if any, for correctness and consistency.

If you have received this message and the following does not apply, please open
an issue at
https://github.com/terraform-providers/terraform-provider-vsphere/issues

!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
`

// diskDeletedName is a placeholder name for deleted disks. This is to assist
// with user-friendliness in the diff.
const diskDeletedName = "<deleted>"

// diskDetachedName is a placeholder name for disks that are getting detached,
// either because they have keep_on_remove set or are external disks attached
// with "attach".
const diskDetachedName = "<remove, keep disk>"

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
			Description: "The datastore ID for this virtual disk, if different than the virtual machine.",
		},
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The file name of the disk. This can be either a name or path relative to the root of the datastore. If simply a name, the disk is located with the virtual machine.",
			ValidateFunc: func(v interface{}, _ string) ([]string, []error) {
				if path.Ext(v.(string)) != ".vmdk" {
					return nil, []error{fmt.Errorf("disk name %s must end in .vmdk", v.(string))}
				}
				return nil, nil
			},
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
			Default:     false,
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
			Default:     false,
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
			Default:      0,
			Description:  "The share count for this disk when the share level is custom.",
			ValidateFunc: validation.IntAtLeast(0),
		},

		// VirtualDisk/Other complex stuff
		"size": {
			Type:         schema.TypeInt,
			Optional:     true,
			Description:  "The size of the disk, in GB.",
			ValidateFunc: validation.IntAtLeast(1),
		},
		"unit_number": {
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      0,
			Description:  "The unique device number for this disk. This number determines where on the SCSI bus this device will be attached.",
			ValidateFunc: validation.IntBetween(0, 59),
		},
		"keep_on_remove": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Set to true to keep the underlying VMDK file when removing this virtual disk from configuration.",
		},
		"attach": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "If this is true, the disk is attached instead of created. Implies keep_on_remove.",
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
func NewDiskSubresource(client *govmomi.Client, rdd resourceDataDiff, d, old map[string]interface{}, idx int) *DiskSubresource {
	sr := &DiskSubresource{
		Subresource: &Subresource{
			schema:  DiskSubresourceSchema(),
			client:  client,
			srtype:  subresourceTypeDisk,
			data:    d,
			olddata: old,
			rdd:     rdd,
		},
	}
	sr.Index = idx
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
	log.Printf("[DEBUG] DiskApplyOperation: Beginning apply operation")
	o, n := d.GetChange(subresourceTypeDisk)
	ods := o.([]interface{})
	nds := n.([]interface{})

	var spec []types.BaseVirtualDeviceConfigSpec

	// Our old and new sets now have an accurate description of devices that may
	// have been added, removed, or changed. Look for removed devices first.
	log.Printf("[DEBUG] DiskApplyOperation: Looking for resources to delete")
nextOld:
	for oi, oe := range ods {
		om := oe.(map[string]interface{})
		for _, ne := range nds {
			nm := ne.(map[string]interface{})
			if nm["name"] == diskDeletedName || nm["name"] == diskDetachedName {
				// This is a "dummy" deleted resource and should be skipped over
				continue
			}
			if om["key"] == nm["key"] {
				continue nextOld
			}
		}
		r := NewDiskSubresource(c, d, om, nil, oi)
		dspec, err := r.Delete(l)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		l = applyDeviceChange(l, dspec)
		spec = append(spec, dspec...)
	}

	// Now check for creates and updates.  The results of this operation are
	// committed to state after the operation completes, on top of the items that
	// have not changed.
	var updates []interface{}
	log.Printf("[DEBUG] DiskApplyOperation: Looking for resources to create or update")
	log.Printf("[DEBUG] DiskApplyOperation: Resources not being changed: %s", subresourceListString(updates))
nextNew:
	for ni, ne := range nds {
		nm := ne.(map[string]interface{})
		if nm["name"] == diskDeletedName || nm["name"] == diskDetachedName {
			// This is a "dummy" deleted resource and should be skipped over
			continue
		}
		for _, oe := range ods {
			om := oe.(map[string]interface{})
			if nm["key"] == om["key"] {
				// This is an update
				r := NewDiskSubresource(c, d, nm, om, ni)
				// If the only thing changing here is the datastore, or keep_on_remove,
				// this is a no-op as far as a device change is concerned. Datastore
				// changes are handled during storage vMotion later on during the
				// update phase. keep_on_remove is a Terraform-only attribute and only
				// needs to be committed to state.
				omc, err := copystructure.Copy(om)
				if err != nil {
					return nil, nil, fmt.Errorf("%s: error generating copy of old disk data: %s", r.Addr(), err)
				}
				omc.(map[string]interface{})["datastore_id"] = nm["datastore_id"]
				omc.(map[string]interface{})["keep_on_remove"] = nm["keep_on_remove"]
				if reflect.DeepEqual(omc, nm) {
					updates = append(updates, r.Data())
					continue nextNew
				}
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
		r := NewDiskSubresource(c, d, nm, nil, ni)
		cspec, err := r.Create(l)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		l = applyDeviceChange(l, cspec)
		spec = append(spec, cspec...)
		updates = append(updates, r.Data())
	}

	log.Printf("[DEBUG] DiskApplyOperation: Post-apply final resource list: %s", subresourceListString(updates))
	// We are now done! Return the updated device list and config spec. Save updates as well.
	if err := d.Set(subresourceTypeDisk, updates); err != nil {
		return nil, nil, err
	}
	log.Printf("[DEBUG] DiskApplyOperation: Device list at end of operation: %s", DeviceListString(l))
	log.Printf("[DEBUG] DiskApplyOperation: Device config operations from apply: %s", DeviceChangeString(spec))
	log.Printf("[DEBUG] DiskApplyOperation: Apply complete, returning updated spec")
	return l, spec, nil
}

// DiskRefreshOperation processes a refresh operation for all of the disks in
// the resource.
//
// This functions similar to DiskApplyOperation, but nothing to change is
// returned, all necessary values are just set and committed to state.
func DiskRefreshOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) error {
	log.Printf("[DEBUG] DiskRefreshOperation: Beginning refresh")
	devices := selectDisks(l, d.Get("scsi_controller_count").(int))
	log.Printf("[DEBUG] DiskRefreshOperation: Disk devices located: %s", DeviceListString(devices))
	curSet := d.Get(subresourceTypeDisk).([]interface{})
	log.Printf("[DEBUG] DiskRefreshOperation: Current resource set from state: %s", subresourceListString(curSet))
	var newSet []interface{}
	// First check for negative keys. These are freshly added devices that are
	// usually coming into read post-create.
	//
	// If we find what we are looking for, we remove the device from the working
	// set so that we don't try and process it in the next few passes.
	log.Printf("[DEBUG] DiskRefreshOperation: Looking for freshly-created or re-assigned resources to read in")
	for i, item := range curSet {
		m := item.(map[string]interface{})
		if m["key"].(int) < 1 {
			r := NewDiskSubresource(c, d, m, nil, i)
			if err := r.Read(l); err != nil {
				return fmt.Errorf("%s: %s", r.Addr(), err)
			}
			if r.Get("key").(int) < 1 {
				// This should not have happened - if it did, our device
				// creation/update logic failed somehow that we were not able to track.
				return fmt.Errorf("device %d with address %s still unaccounted for after update/read", r.Get("key").(int), r.Get("device_address").(string))
			}
			newSet = append(newSet, r.Data())
			for i := 0; i < len(devices); i++ {
				device := devices[i]
				if device.GetVirtualDevice().Key == int32(r.Get("key").(int)) {
					devices = append(devices[:i], devices[i+1:]...)
					i--
				}
			}
		}
	}
	log.Printf("[DEBUG] DiskRefreshOperation: Disk devices after created/re-assigned device search: %s", DeviceListString(devices))
	log.Printf("[DEBUG] DiskRefreshOperation: Resource set to write after created/re-assigned device search: %s", subresourceListString(newSet))

	// Go over the remaining devices, refresh via key, and then remove their
	// entries as well.
	log.Printf("[DEBUG] DiskRefreshOperation: Looking for devices known in state")
	for i := 0; i < len(devices); i++ {
		device := devices[i]
		for n, item := range curSet {
			m := item.(map[string]interface{})
			if m["key"].(int) < 1 {
				// Skip any of these keys as we won't be matching any of those anyway here
				continue
			}
			if device.GetVirtualDevice().Key != int32(m["key"].(int)) {
				// Skip any device that doesn't match key as well
				continue
			}
			// We should have our device -> resource match, so read now.
			r := NewDiskSubresource(c, d, m, nil, n)
			if err := r.Read(l); err != nil {
				return fmt.Errorf("%s: %s", r.Addr(), err)
			}
			// Done reading, push this onto our new set and remove the device from
			// the list
			newSet = append(newSet, r.Data())
			devices = append(devices[:i], devices[i+1:]...)
			i--
		}
	}
	log.Printf("[DEBUG] DiskRefreshOperation: Resource set to write after known device search: %s", subresourceListString(newSet))
	log.Printf("[DEBUG] DiskRefreshOperation: Probable orphaned disk devices: %s", DeviceListString(devices))

	// Finally, any device that is still here is orphaned. They should be added
	// as new devices.
	log.Printf("[DEBUG] DiskRefreshOperation: Adding orphaned devices")
	for _, device := range devices {
		m := make(map[string]interface{})
		vd := device.GetVirtualDevice()
		ctlr := l.FindByKey(vd.ControllerKey)
		if ctlr == nil {
			return fmt.Errorf("could not find controller with key %d", vd.Key)
		}
		m["key"] = int(vd.Key)
		var err error
		m["device_address"], err = computeDevAddr(vd, ctlr.(types.BaseVirtualController))
		if err != nil {
			return fmt.Errorf("error computing device address: %s", err)
		}
		// We want to set keep_on_remove for these disks as well so that they are
		// not destroyed when we remove them in the next TF run.
		m["keep_on_remove"] = true
		r := NewDiskSubresource(c, d, m, nil, len(newSet))
		if err := r.Read(l); err != nil {
			return fmt.Errorf("%s: %s", r.Addr(), err)
		}
		newSet = append(newSet, r.Data())
	}

	log.Printf("[DEBUG] DiskRefreshOperation: Resource set to write after adding orphaned devices: %s", subresourceListString(newSet))
	// Sort the device list by unit number. This provides some semblance of order
	// in the state as devices are added and removed.
	sort.Sort(virtualDiskSubresourceSorter(newSet))
	log.Printf("[DEBUG] DiskRefreshOperation: Final (sorted) resource set to write: %s", subresourceListString(newSet))
	log.Printf("[DEBUG] DiskRefreshOperation: Refresh operation complete, sending new resource set")
	return d.Set(subresourceTypeDisk, newSet)
}

// DiskDestroyOperation process the destroy operation for virtual disks.
//
// Disks are the only real operation that require special destroy logic, and
// that's because we want to check to make sure that we detach any disks that
// need to be simply detached (not deleted) before we destroy the entire
// virtual machine, as that would take those disks with it.
func DiskDestroyOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	log.Printf("[DEBUG] DiskDestroyOperation: Beginning destroy")
	// All we are doing here is getting a config spec for detaching the disks
	// that we need to detach, so we don't need the vast majority of the stateful
	// logic that is in deviceApplyOperation.
	ds := d.Get(subresourceTypeDisk).([]interface{})

	var spec []types.BaseVirtualDeviceConfigSpec

	log.Printf("[DEBUG] DiskDestroyOperation: Detaching devices with keep_on_remove enabled")
	for oi, oe := range ds {
		m := oe.(map[string]interface{})
		if !m["keep_on_remove"].(bool) && !m["attach"].(bool) {
			// We don't care about disks we haven't set to keep
			continue
		}
		r := NewDiskSubresource(c, d, m, nil, oi)
		dspec, err := r.Delete(l)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		applyDeviceChange(l, dspec)
		spec = append(spec, dspec...)
	}

	log.Printf("[DEBUG] DiskDestroyOperation: Device config operations from destroy: %s", DeviceChangeString(spec))
	return spec, nil
}

// DiskDiffOperation performs operations relevant to managing the diff on disk
// sub-resources.
//
// Most importantly, this works to prevent spurious diffs by extrapolating the
// correct correlation between the old and new sets using the name as a primary
// key, and then normalizing the two diffs so that computed data is properly
// set.
//
// The following validation operations are also carried out on the set as a
// whole:
//
// * Ensuring all names are unique across the set.
// * Ensuring that at least one element in the set has a unit_number of 0.
func DiskDiffOperation(d *schema.ResourceDiff, c *govmomi.Client) error {
	log.Printf("[DEBUG] DiskDiffOperation: Beginning disk diff customization")
	o, n := d.GetChange(subresourceTypeDisk)
	// Some global validation first. We handle individual validation later.
	log.Printf("[DEBUG] DiskDiffOperation: Beginning diff validation (indexes aligned to new config)")
	names := make(map[string]struct{})
	units := make(map[int]struct{})
	if len(n.([]interface{})) < 1 {
		return errors.New("there must be at least one disk specified")
	}
	for ni, ne := range n.([]interface{}) {
		nm := ne.(map[string]interface{})
		// Because we support long and short-form paths, we don't support duplicate
		// file names right now, even if they are in a different path. This makes
		// things slightly more inflexible but hopefully not enough to be seriously
		// cumbersome to the use of the resource.
		name := path.Base(nm["name"].(string))
		if _, ok := names[name]; ok {
			return fmt.Errorf("disk: duplicate name %s", name)
		}
		if _, ok := units[nm["unit_number"].(int)]; ok {
			return fmt.Errorf("disk: duplicate unit_number %d", nm["unit_number"].(int))
		}
		names[name] = struct{}{}
		units[nm["unit_number"].(int)] = struct{}{}
		// Run the resource through an individual validate function. This performs
		// field validation for things we don't need to know the state of other
		// resources for.
		r := NewDiskSubresource(c, d, nm, nil, ni)
		if err := r.ValidateDiff(); err != nil {
			return fmt.Errorf("%s: %s", r.Addr(), err)
		}
	}
	if _, ok := units[0]; !ok {
		return errors.New("at least one disk must have a unit_number of 0")
	}

	// Perform the normalization here.
	log.Printf("[DEBUG] DiskDiffOperation: Beginning diff normalization (indexes aligned to old state)")
	ods := o.([]interface{})
	nds := n.([]interface{})

	normalized := make([]interface{}, len(ods))
nextNew:
	for _, ne := range nds {
		nm := ne.(map[string]interface{})
		for oi, oe := range ods {
			om := oe.(map[string]interface{})
			// We extrapolate using the name as a "primary key" of sorts. Since we
			// support both long-form and short-form paths, and don't support using
			// same file name regardless of if you are using a long-from path, we
			// just check the short-form and do the comparison from there.
			if path.Base(nm["name"].(string)) == path.Base(om["name"].(string)) {
				r := NewDiskSubresource(c, d, nm, om, oi)
				if err := r.NormalizeDiff(); err != nil {
					return fmt.Errorf("%s: %s", r.Addr(), err)
				}
				normalized[oi] = r.Data()
				continue nextNew
			}
		}
		// We didn't find a match for this resource, it could be a new resource or
		// significantly altered. Put it back on the list in the same form we got
		// it in, but delete the key and device address first, just in case it was
		// in a position previously occupied by an existing resource.
		nm["key"] = 0
		nm["device_address"] = ""
		normalized = append(normalized, nm)
	}

	// Go thru the new list, and replace any nils with a "deleted" copy of the
	// old resource. This is basically a copy of the old entry with <deleted> in
	// place of the name, so it shows up nicely in the diff.
	for ni, ne := range normalized {
		if ne != nil {
			continue
		}
		nv, err := copystructure.Copy(ods[ni])
		if err != nil {
			return fmt.Errorf("disk.%d: error making updated diff of deleted entry: %s", ni, err)
		}
		nm := nv.(map[string]interface{})
		switch {
		case nm["keep_on_remove"].(bool):
			fallthrough
		case nm["attach"].(bool):
			nm["name"] = diskDetachedName
		default:
			nm["name"] = diskDeletedName
		}
		normalized[ni] = nm
	}

	// All done. We can end the customization off by setting the new, normalized diff.
	log.Printf("[DEBUG] DiskDiffOperation: New resource set post-normalization: %s", subresourceListString(normalized))
	log.Printf("[DEBUG] DiskDiffOperation: Disk diff customization complete, sending new diff")
	return d.SetNew(subresourceTypeDisk, normalized)
}

// DiskCloneValidateOperation takes the VirtualDeviceList, which should come
// from a source VM or template, and validates the following:
//
// * There are at least as many disks defined in the configuration as there are
// in the source VM or template.
// * All disks survive a disk sub-resource read operation.
//
// This function is meant to be called during diff customization. It is a
// subset of the normal refresh behaviour as we don't worry about checking
// existing state.
func DiskCloneValidateOperation(d *schema.ResourceDiff, c *govmomi.Client, l object.VirtualDeviceList, linked bool) error {
	log.Printf("[DEBUG] DiskCloneValidateOperation: Checking existing virtual disk configuration")
	devices := selectDisks(l, d.Get("scsi_controller_count").(int))
	// Sort the device list, in case it's not sorted already.
	devSort := virtualDeviceListSorter{
		Sort:       devices,
		DeviceList: l,
	}
	log.Printf("[DEBUG] DiskCloneValidateOperation: Disk devices order before sort: %s", DeviceListString(devices))
	sort.Sort(devSort)
	devices = devSort.Sort
	log.Printf("[DEBUG] DiskCloneValidateOperation: Disk devices order after sort: %s", DeviceListString(devices))
	// Do the same for our listed disks.
	curSet := d.Get(subresourceTypeDisk).([]interface{})
	log.Printf("[DEBUG] DiskCloneValidateOperation: Current resource set: %s", subresourceListString(curSet))
	sort.Sort(virtualDiskSubresourceSorter(curSet))
	log.Printf("[DEBUG] DiskCloneValidateOperation: Resource set order after sort: %s", subresourceListString(curSet))

	// Quickly validate length. If there are more disks in the template than
	// there is in the configuration, kick out an error.
	if len(devices) > len(curSet) {
		return fmt.Errorf("not enough disks in configuration - you need at least %d to use this template (current: %d)", len(devices), len(curSet))
	}

	// Do test read operations on all disks.
	log.Printf("[DEBUG] DiskCloneValidateOperation: Running test read operations on all disks")
	for i, device := range devices {
		m := make(map[string]interface{})
		vd := device.GetVirtualDevice()
		ctlr := l.FindByKey(vd.ControllerKey)
		if ctlr == nil {
			return fmt.Errorf("could not find controller with key %d", vd.Key)
		}
		m["key"] = int(vd.Key)
		var err error
		m["device_address"], err = computeDevAddr(vd, ctlr.(types.BaseVirtualController))
		if err != nil {
			return fmt.Errorf("error computing device address: %s", err)
		}
		r := NewDiskSubresource(c, d, m, nil, i)
		if err := r.Read(l); err != nil {
			return fmt.Errorf("%s: validation failed (%s)", r.Addr(), err)
		}
		// Load the target resource to do a few comparisons for correctness in config.
		targetM := curSet[i].(map[string]interface{})
		tr := NewDiskSubresource(c, d, targetM, nil, i)
		// Ensure that the file names match. vSphere does not allow you to choose
		// the name of existing disks during a clone and will rename them according
		// to the standard convention of <name>.vmdk, <name>_1.vmdk, etc.  Hence we
		// need to enforce this on all created VMs as well.
		name := d.Get("name").(string)
		var extra string
		if i > 0 {
			extra = fmt.Sprintf("_%d", i)
		}
		expected := fmt.Sprintf("%s%s.vmdk", name, extra)
		if tr.Get("name").(string) != expected {
			return fmt.Errorf("%s: invalid disk name %q for cloning. Please rename this disk to %q", tr.Addr(), tr.Get("name").(string), expected)
		}

		// Do some pre-clone validation. This is mainly to make sure that the disks
		// clone in a way that is consistent with configuration.
		targetName := tr.Get("name").(string)
		sourceSize := r.Get("size").(int)
		targetSize := tr.Get("size").(int)
		targetThin := tr.Get("thin_provisioned").(bool)
		targetEager := tr.Get("eagerly_scrub").(bool)

		var sourceThin, sourceEager bool
		if b := r.Get("thin_provisioned"); b != nil {
			sourceThin = b.(bool)
		}
		if b := r.Get("eagerly_scrub"); b != nil {
			sourceEager = b.(bool)
		}

		switch {
		case linked:
			switch {
			case sourceSize != targetSize:
				return fmt.Errorf("%s: disk name %s must be the exact size of source when using linked_clone (expected: %d GiB)", tr.Addr(), targetName, sourceSize)
			case sourceThin != targetThin:
				return fmt.Errorf("%s: disk name %s must have same value for thin_provisioned as source when using linked_clone (expected: %t)", tr.Addr(), targetName, sourceThin)
			case sourceEager != targetEager:
				return fmt.Errorf("%s: disk name %s must have same value for eagerly_scrub as source when using linked_clone (expected: %t)", tr.Addr(), targetName, sourceEager)
			}
		default:
			if sourceSize > targetSize {
				return fmt.Errorf("%s: disk name %s must be at least the same size of source when cloning (expected: >= %d GiB)", tr.Addr(), targetName, sourceSize)
			}
		}

		// Finally, we don't support non-SCSI (ie: SATA, IDE, NVMe) disks, so kick
		// back an error if we see one of those.
		ct, _, _, err := splitDevAddr(r.DevAddr())
		if err != nil {
			return fmt.Errorf("%s: error parsing device address after reading disk %q: %s", tr.Addr(), r.Get("name").(string), err)
		}
		if ct != SubresourceControllerTypeSCSI {
			return fmt.Errorf("%s: unsupported controller type %s for disk %q. Please use a template with SCSI disks only", tr.Addr(), ct, tr.Get("name").(string))
		}
	}
	log.Printf("[DEBUG] DiskCloneValidateOperation: All disks in source validated successfully")
	return nil
}

// DiskMigrateRelocateOperation assembles the
// VirtualMachineRelocateSpecDiskLocator slice for a virtual machine migration
// operation, otherwise known as storage vMotion.
func DiskMigrateRelocateOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) ([]types.VirtualMachineRelocateSpecDiskLocator, error) {
	log.Printf("[DEBUG] DiskMigrateRelocateOperation: Generating any necessary disk relocate specs")
	ods, nds := d.GetChange(subresourceTypeDisk)

	var relocators []types.VirtualMachineRelocateSpecDiskLocator

	// We are only concerned with resources that would normally be updated, as
	// incoming or outgoing disks obviously won't need migrating. Hence, this is
	// a simplified subset of the normal apply logic.
nextDisk:
	for ni, ne := range nds.([]interface{}) {
		nm := ne.(map[string]interface{})
		if nm["name"] == diskDeletedName || nm["name"] == diskDetachedName {
			continue
		}
		for _, oe := range ods.([]interface{}) {
			om := oe.(map[string]interface{})
			if nm["key"] == om["key"] {
				// No change in datastore is a no-op, unless we are changing default datastores
				if nm["datastore_id"] == om["datastore_id"] && !d.HasChange("datastore_id") {
					continue nextDisk
				}
				r := NewDiskSubresource(c, d, nm, om, ni)
				relocator, err := r.Relocate(l)
				if err != nil {
					return nil, fmt.Errorf("%s: %s", r.Addr(), err)
				}
				if d.Get("datastore_id").(string) == relocator.Datastore.Value {
					log.Printf("[DEBUG] %s: Datastore in spec is same as default, dropping in favor of implicit relocation", r.Addr())
					continue nextDisk
				}
				relocators = append(relocators, relocator)
			}
		}
	}

	log.Printf("[DEBUG] DiskMigrateRelocateOperation: Disk relocator list: %s", diskRelocateListString(relocators))
	log.Printf("[DEBUG] DiskMigrateRelocateOperation: Disk relocator generation complete")
	return relocators, nil
}

// DiskCloneRelocateOperation assembles the
// VirtualMachineRelocateSpecDiskLocator slice for a virtual machine clone
// operation.
//
// This differs from a regular storage vMotion in that we have no existing
// devices in the resource to work off of - the disks in the source virtual
// machine is purely our source of truth. These disks are assigned to our disk
// sub-resources in config and the relocate specs are generated off of the
// filename and backing data defined in config, taking on these filenames when
// cloned. After the clone is complete, natural re-configuration happens to
// bring the disk configurations fully in sync with that is defined.
func DiskCloneRelocateOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) ([]types.VirtualMachineRelocateSpecDiskLocator, error) {
	log.Printf("[DEBUG] DiskCloneRelocateOperation: Generating full disk relocate spec list")
	devices := selectDisks(l, d.Get("scsi_controller_count").(int))
	log.Printf("[DEBUG] DiskCloneRelocateOperation: Disk devices located: %s", DeviceListString(devices))
	// Sort the device list, in case it's not sorted already.
	devSort := virtualDeviceListSorter{
		Sort:       devices,
		DeviceList: l,
	}
	sort.Sort(devSort)
	devices = devSort.Sort
	log.Printf("[DEBUG] DiskCloneRelocateOperation: Disk devices order after sort: %s", DeviceListString(devices))
	// Do the same for our listed disks.
	curSet := d.Get(subresourceTypeDisk).([]interface{})
	log.Printf("[DEBUG] DiskCloneRelocateOperation: Current resource set: %s", subresourceListString(curSet))
	sort.Sort(virtualDiskSubresourceSorter(curSet))
	log.Printf("[DEBUG] DiskCloneRelocateOperation: Resource set order after sort: %s", subresourceListString(curSet))

	log.Printf("[DEBUG] DiskCloneRelocateOperation: Generating relocators for source disks")
	var relocators []types.VirtualMachineRelocateSpecDiskLocator
	for i, device := range devices {
		m := curSet[i].(map[string]interface{})
		vd := device.GetVirtualDevice()
		ctlr := l.FindByKey(vd.ControllerKey)
		if ctlr == nil {
			return nil, fmt.Errorf("could not find controller with key %d", vd.Key)
		}
		m["key"] = int(vd.Key)
		var err error
		m["device_address"], err = computeDevAddr(vd, ctlr.(types.BaseVirtualController))
		if err != nil {
			return nil, fmt.Errorf("error computing device address: %s", err)
		}
		r := NewDiskSubresource(c, d, m, nil, i)
		relocator, err := r.Relocate(l)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		relocators = append(relocators, relocator)
	}

	log.Printf("[DEBUG] DiskCloneRelocateOperation: Disk relocator list: %s", diskRelocateListString(relocators))
	log.Printf("[DEBUG] DiskCloneRelocateOperation: Disk relocator generation complete")
	return relocators, nil
}

// DiskPostCloneOperation normalizes the virtual disks on a freshly-cloned
// virtual machine and outputs any necessary device change operations. It also
// sets the state in advance of the post-create read.
//
// This differs from a regular apply operation in that a configuration is
// already present, but we don't have any existing state, which the standard
// virtual device operations rely pretty heavily on.
func DiskPostCloneOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) (object.VirtualDeviceList, []types.BaseVirtualDeviceConfigSpec, error) {
	log.Printf("[DEBUG] DiskPostCloneOperation: Looking for disk device changes post-clone")
	devices := selectDisks(l, d.Get("scsi_controller_count").(int))
	log.Printf("[DEBUG] DiskPostCloneOperation: Disk devices located: %s", DeviceListString(devices))
	// Sort the device list, in case it's not sorted already.
	devSort := virtualDeviceListSorter{
		Sort:       devices,
		DeviceList: l,
	}
	sort.Sort(devSort)
	devices = devSort.Sort
	log.Printf("[DEBUG] DiskPostCloneOperation: Disk devices order after sort: %s", DeviceListString(devices))
	// Do the same for our listed disks.
	curSet := d.Get(subresourceTypeDisk).([]interface{})
	log.Printf("[DEBUG] DiskPostCloneOperation: Current resource set: %s", subresourceListString(curSet))
	sort.Sort(virtualDiskSubresourceSorter(curSet))
	log.Printf("[DEBUG] DiskPostCloneOperation: Resource set order after sort: %s", subresourceListString(curSet))

	var spec []types.BaseVirtualDeviceConfigSpec
	var updates []interface{}

	log.Printf("[DEBUG] DiskPostCloneOperation: Looking for and applying device changes in source disks")
	for i, device := range devices {
		src := curSet[i].(map[string]interface{})
		vd := device.GetVirtualDevice()
		ctlr := l.FindByKey(vd.ControllerKey)
		if ctlr == nil {
			return nil, nil, fmt.Errorf("could not find controller with key %d", vd.Key)
		}
		src["key"] = int(vd.Key)
		var err error
		src["device_address"], err = computeDevAddr(vd, ctlr.(types.BaseVirtualController))
		if err != nil {
			return nil, nil, fmt.Errorf("error computing device address: %s", err)
		}
		// Copy the source set into old. This allows us to patch a copy of the
		// product of this set with the source, creating a diff.
		old, err := copystructure.Copy(src)
		if err != nil {
			return nil, nil, fmt.Errorf("error copying source set for disk at unit_number %d: %s", src["unit_number"].(int), err)
		}
		rOld := NewDiskSubresource(c, d, old.(map[string]interface{}), nil, i)
		if err := rOld.Read(l); err != nil {
			return nil, nil, fmt.Errorf("%s: %s", rOld.Addr(), err)
		}
		new, err := copystructure.Copy(rOld.Data())
		if err != nil {
			return nil, nil, fmt.Errorf("error copying current device state for disk at unit_number %d: %s", src["unit_number"].(int), err)
		}
		for k, v := range src {
			// Skip name, datastore_id, and share count if share level isn't custom
			switch k {
			case "name", "datastore_id":
				continue
			case "io_share_count":
				if src["io_share_level"] != string(types.SharesLevelCustom) {
					continue
				}
			}
			new.(map[string]interface{})[k] = v
		}
		rNew := NewDiskSubresource(c, d, new.(map[string]interface{}), rOld.Data(), i)
		if !reflect.DeepEqual(rNew.Data(), rOld.Data()) {
			uspec, err := rNew.Update(l)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %s", rNew.Addr(), err)
			}
			l = applyDeviceChange(l, uspec)
			spec = append(spec, uspec...)
		}
		updates = append(updates, rNew.Data())
	}

	// Any disk past the current device list is a new device. Create those now.
	for _, ni := range curSet[len(devices):] {
		r := NewDiskSubresource(c, d, ni.(map[string]interface{}), nil, len(updates))
		cspec, err := r.Create(l)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		l = applyDeviceChange(l, cspec)
		spec = append(spec, cspec...)
		updates = append(updates, r.Data())
	}

	log.Printf("[DEBUG] DiskPostCloneOperation: Post-clone final resource list: %s", subresourceListString(updates))
	if err := d.Set(subresourceTypeDisk, updates); err != nil {
		return nil, nil, err
	}
	log.Printf("[DEBUG] DiskPostCloneOperation: Device list at end of operation: %s", DeviceListString(l))
	log.Printf("[DEBUG] DiskPostCloneOperation: Device config operations from post-clone: %s", DeviceChangeString(spec))
	log.Printf("[DEBUG] DiskPostCloneOperation: Operation complete, returning updated spec")
	return l, spec, nil
}

// DiskImportOperation validates the disk configuration of the virtual
// machine's VirtualDeviceList to ensure it will be imported properly.
func DiskImportOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) error {
	log.Printf("[DEBUG] DiskImportOperation: Performing disk import validation")
	devices := selectDisks(l, d.Get("scsi_controller_count").(int))
	log.Printf("[DEBUG] DiskImportOperation: Disk devices located: %s", DeviceListString(devices))

	// Read in the disks. We don't do anything with the results here other than
	// validate that the disks are SCSI disks. The read operation validates the rest.
	log.Printf("[DEBUG] DiskImportOperation: Validating disks")
	for i, device := range devices {
		m := make(map[string]interface{})
		vd := device.GetVirtualDevice()
		ctlr := l.FindByKey(vd.ControllerKey)
		if ctlr == nil {
			return fmt.Errorf("could not find controller with key %d", vd.Key)
		}
		m["key"] = int(vd.Key)
		var err error
		m["device_address"], err = computeDevAddr(vd, ctlr.(types.BaseVirtualController))
		if err != nil {
			return fmt.Errorf("error computing device address: %s", err)
		}
		r := NewDiskSubresource(c, d, m, nil, i)
		if err := r.Read(l); err != nil {
			return fmt.Errorf("%s: %s", r.Addr(), err)
		}
		ct, _, _, err := splitDevAddr(r.DevAddr())
		if err != nil {
			return fmt.Errorf("%s: error parsing device address after reading disk %q: %s", r.Addr(), r.Get("name").(string), err)
		}
		if ct != SubresourceControllerTypeSCSI {
			return fmt.Errorf("%s: unsupported controller type %s for disk %q. The VM resource supports SCSI disks only", r.Addr(), ct, r.Get("name").(string))
		}
	}
	log.Printf("[DEBUG] DiskImportOperation: Disk validation complete")
	return nil
}

// ReadDiskAttrsForDataSource returns select attributes from the list of disks
// on a virtual machine. This is used in the VM data source to discover
// specific options of all of the disks on the virtual machine sorted by the
// order that they would be added in if a clone were to be done.
func ReadDiskAttrsForDataSource(l object.VirtualDeviceList, count int) ([]map[string]interface{}, error) {
	log.Printf("[DEBUG] ReadDiskAttrsForDataSource: Fetching select attributes for disks across %d SCSI controllers", count)
	devices := selectDisks(l, count)
	log.Printf("[DEBUG] ReadDiskAttrsForDataSource: Disk devices located: %s", DeviceListString(devices))
	// Sort the device list, in case it's not sorted already.
	devSort := virtualDeviceListSorter{
		Sort:       devices,
		DeviceList: l,
	}
	sort.Sort(devSort)
	devices = devSort.Sort
	log.Printf("[DEBUG] ReadDiskAttrsForDataSource: Disk devices order after sort: %s", DeviceListString(devices))
	var out []map[string]interface{}
	for i, device := range devices {
		disk := device.(*types.VirtualDisk)
		backing, ok := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo)
		if !ok {
			return nil, fmt.Errorf("disk number %d has an unsupported backing type (expected flat VMDK version 2, got %T", i, disk.Backing)
		}
		m := make(map[string]interface{})
		var eager, thin bool
		if backing.EagerlyScrub != nil {
			eager = *backing.EagerlyScrub
		}
		if backing.ThinProvisioned != nil {
			thin = *backing.ThinProvisioned
		}
		m["size"] = int(structure.ByteToGiB(disk.CapacityInBytes).(int64))
		m["eagerly_scrub"] = eager
		m["thin_provisioned"] = thin
		out = append(out, m)
	}
	log.Printf("[DEBUG] ReadDiskAttrsForDataSource: Attributes returned: %+v", out)
	return out, nil
}

// Create creates a vsphere_virtual_machine disk sub-resource.
func (r *DiskSubresource) Create(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	log.Printf("[DEBUG] %s: Creating disk", r)
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
	if err := r.SaveDevIDs(disk, ctlr); err != nil {
		return nil, err
	}
	dspec, err := object.VirtualDeviceList{disk}.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
	if err != nil {
		return nil, err
	}
	if len(dspec) != 1 {
		return nil, fmt.Errorf("incorrect number of config spec items returned - expected 1, got %d", len(dspec))
	}
	// Clear the file operation if we are attaching.
	if r.Get("attach").(bool) {
		dspec[0].GetVirtualDeviceConfigSpec().FileOperation = ""
	}
	spec = append(spec, dspec...)
	log.Printf("[DEBUG] %s: Device config operations from create: %s", r, DeviceChangeString(spec))
	log.Printf("[DEBUG] %s: Create finished", r)
	return spec, nil
}

// Read reads a vsphere_virtual_machine disk sub-resource and commits the data
// to the newData layer.
func (r *DiskSubresource) Read(l object.VirtualDeviceList) error {
	log.Printf("[DEBUG] %s: Reading state", r)
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
	if err := r.SaveDevIDs(disk, ctlr); err != nil {
		return err
	}

	// Fetch disk attachment state in config
	var attach bool
	if r.Get("attach") != nil {
		attach = r.Get("attach").(bool)
	}
	// Save disk backing settings
	b, ok := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo)
	if !ok {
		return fmt.Errorf("disk backing at %s is of an unsupported type (type %T)", r.Get("device_address").(string), disk.Backing)
	}
	r.Set("disk_mode", b.DiskMode)
	r.Set("write_through", b.WriteThrough)

	// Only use disk_sharing if we are on vSphere 6.0 and higher
	version := viapi.ParseVersionFromClient(r.client)
	if version.Newer(viapi.VSphereVersion{Product: version.Product, Major: 6}) {
		r.Set("disk_sharing", b.Sharing)
	}

	if !attach {
		r.Set("thin_provisioned", b.ThinProvisioned)
		r.Set("eagerly_scrub", b.EagerlyScrub)
	}
	r.Set("datastore_id", b.Datastore.Value)

	// Walk up the child disk path until we have a path that matches our actual
	// disk name. If there is no disk name, of if we can't find a match, just
	// save the main backing.
	origName := r.Get("name")
	var name string
	if origName != nil && origName.(string) != "" && !datastorePathHasBase(b.FileName, origName.(string)) && b.Parent != nil {
		name = walkDiskBacking(b.Parent, origName.(string))
	}
	if name == "" {
		name = b.FileName
	}
	dp := &object.DatastorePath{}
	if ok := dp.FromString(name); !ok {
		return fmt.Errorf("could not parse path from filename: %s", b.FileName)
	}

	// Validate that our names match on the base. If they don't, something is seriously wrong.
	if origName != nil && origName.(string) != "" {
		ob := path.Base(origName.(string))
		cb := path.Base(dp.Path)
		if ob != cb {
			return fmt.Errorf(
				diskReadNameMismatchError,
				r.rdd.Get("name").(string),
				r.Get("key").(int),
				r.Get("device_address").(string),
				ob,
				cb,
			)
		}
	}
	r.Set("name", dp.Path)

	// Disk settings
	if !attach {
		r.Set("size", structure.ByteToGiB(disk.CapacityInBytes))
	}

	if allocation := disk.StorageIOAllocation; allocation != nil {
		r.Set("io_limit", allocation.Limit)
		r.Set("io_reservation", allocation.Reservation)
		if shares := allocation.Shares; shares != nil {
			r.Set("io_share_level", string(shares.Level))
			r.Set("io_share_count", shares.Shares)
		}
	}
	log.Printf("[DEBUG] %s: Read finished (key and device address may have changed)", r)
	return nil
}

// Update updates a vsphere_virtual_machine disk sub-resource.
func (r *DiskSubresource) Update(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	log.Printf("[DEBUG] %s: Beginning update", r)
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
		r.SetRestart("unit_number")
		if err := r.SaveDevIDs(disk, ctlr); err != nil {
			return nil, fmt.Errorf("error saving device address: %s", err)
		}
		// A change in disk unit number forces a device key change after the
		// reconfigure. We need to keep the key in the device change spec we send
		// along, but we can reset it here safely. Set it to 0, which will send it
		// though the new device loop, but will distinguish it from newly-created
		// devices.
		r.Set("key", 0)
	}

	// We can now expand the rest of the settings.
	if err := r.expandDiskSettings(disk); err != nil {
		return nil, err
	}

	dspec, err := object.VirtualDeviceList{disk}.ConfigSpec(types.VirtualDeviceConfigSpecOperationEdit)
	if err != nil {
		return nil, err
	}
	if len(dspec) != 1 {
		return nil, fmt.Errorf("incorrect number of config spec items returned - expected 1, got %d", len(dspec))
	}
	// Clear file operation - VirtualDeviceList currently sets this to replace, which is invalid
	dspec[0].GetVirtualDeviceConfigSpec().FileOperation = ""
	log.Printf("[DEBUG] %s: Device config operations from update: %s", r, DeviceChangeString(dspec))
	log.Printf("[DEBUG] %s: Update complete", r)
	return dspec, nil
}

// Delete deletes a vsphere_virtual_machine disk sub-resource.
func (r *DiskSubresource) Delete(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	log.Printf("[DEBUG] %s: Beginning delete", r)
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
	if len(deleteSpec) != 1 {
		return nil, fmt.Errorf("incorrect number of config spec items returned - expected 1, got %d", len(deleteSpec))
	}
	if r.Get("keep_on_remove").(bool) || r.Get("attach").(bool) {
		// Clear file operation so that the disk is kept on remove.
		deleteSpec[0].GetVirtualDeviceConfigSpec().FileOperation = ""
	}
	log.Printf("[DEBUG] %s: Device config operations from update: %s", r, DeviceChangeString(deleteSpec))
	log.Printf("[DEBUG] %s: Delete completed", r)
	return deleteSpec, nil
}

// ValidateDiff performs any complex validation of an individual disk
// sub-resource that can't be done in schema alone.
//
// Do not use resourceData in this function as it's not populated and calls to
// it will cause a panic.
func (r *DiskSubresource) ValidateDiff() error {
	log.Printf("[DEBUG] %s: Beginning diff validation (device information may be incomplete)", r)
	name := r.Get("name").(string)

	// Enforce the maximum unit number, which is the current value of
	// scsi_controller_count * 15 - 1.
	ctlrCount := r.rdd.Get("scsi_controller_count").(int)
	maxUnit := ctlrCount*15 - 1
	currentUnit := r.Get("unit_number").(int)
	if currentUnit > maxUnit {
		return fmt.Errorf("unit_number on disk %q too high (%d) - maximum value is %d with %d SCSI controller(s)", name, currentUnit, maxUnit, ctlrCount)
	}

	switch r.Get("attach").(bool) {
	case true:
		switch {
		case r.Get("datastore_id").(string) == "":
			return fmt.Errorf("datastore_id for disk %q is required when attach is set", name)
		case r.Get("size").(int) > 0:
			return fmt.Errorf("size for disk %q cannot be defined when attach is set", name)
		case r.Get("eagerly_scrub").(bool):
			return fmt.Errorf("eagerly_scrub for disk %q cannot be defined when attach is set", name)
		case r.Get("keep_on_remove").(bool):
			return fmt.Errorf("keep_on_remove for disk %q is implicit when attach is set, please remove this setting", name)
		}
	default:
		if r.Get("size").(int) < 1 {
			return fmt.Errorf("size for disk %q: required option not set", name)
		}
	}
	log.Printf("[DEBUG] %s: Diff validation complete", r)
	return nil
}

// NormalizeDiff checks the diff for a vsphere_virtual_machine disk
// sub-resource. It should be called after setting data and olddata to values
// that seems similar enough to be possibly the same resource, after which this
// function should then compare them and fill in any values that may be missing
// from the new set due to computed values, such as inferred datastore or path
// names.
//
// Do not use resourceData in this function as it's not populated and calls to
// it will cause a panic.
func (r *DiskSubresource) NormalizeDiff() error {
	log.Printf("[DEBUG] %s: Beginning diff normalization (device information may be incomplete)", r)
	// key and device_address will always be non-populated, so copy those.
	okey, _ := r.GetChange("key")
	odaddr, _ := r.GetChange("device_address")
	r.Set("key", okey)
	r.Set("device_address", odaddr)
	// Set the datastore if it's missing as we infer this from the default
	// datastore in that case
	if r.Get("datastore_id") == "" {
		switch {
		case r.rdd.HasChange("datastore_id"):
			// If the default datastore is changing and we don't have a default
			// datastore here, we need to use the implicit setting here to indicate
			// that we may need to migrate. This allows us to differentiate between a
			// full storage vMotion no-op, an implicit migration, and a migration
			// where we will need to generate a relocate spec for the individual disk
			// to ensure it stays at a datastore it might be pinned on.
			r.Set("datastore_id", r.rdd.Get("datastore_id"))
		default:
			odsid, _ := r.GetChange("datastore_id")
			r.Set("datastore_id", odsid)
		}
	}
	// Preserve the share value if we don't have custom shares set
	osc, _ := r.GetChange("io_share_count")
	if r.Get("io_share_level").(string) != string(types.SharesLevelCustom) {
		r.Set("io_share_count", osc)
	}
	// Normalize the path. This should have already have been vetted as being
	// ultimately the same path by the caller.
	oname, _ := r.GetChange("name")
	r.Set("name", oname)

	// Ensure that the user is not attempting to shrink the disk. If we do more
	// we might want to change the name of this method, but we want to check this
	// here as CustomizeDiff is meant for vetoing.
	name := r.Get("name").(string)

	osize, nsize := r.GetChange("size")
	if osize.(int) > nsize.(int) {
		return fmt.Errorf("virtual disk %q: virtual disks cannot be shrunk (old: %d new: %d)", name, osize.(int), nsize.(int))
	}

	// Ensure that there is no change in either eagerly_scrub or thin_provisioned
	// - these values cannot be changed once set.
	if _, err := r.GetWithVeto("eagerly_scrub"); err != nil {
		return fmt.Errorf("virtual disk %q: %s", name, err)
	}
	if _, err := r.GetWithVeto("thin_provisioned"); err != nil {
		return fmt.Errorf("virtual disk %q: %s", name, err)
	}

	// Block certain options from being set depending on the vSphere version.
	version := viapi.ParseVersionFromClient(r.client)
	if r.Get("disk_sharing").(string) != string(types.VirtualDiskSharingSharingNone) {
		if version.Older(viapi.VSphereVersion{Product: version.Product, Major: 6}) {
			return fmt.Errorf("multi-writer disk_sharing is only supported on vSphere 6 and higher")
		}
	}

	log.Printf("[DEBUG] %s: Diff normalization complete", r)
	return nil
}

// Relocate produces a VirtualMachineRelocateSpecDiskLocator for this resource
// and is used for both cloning and storage vMotion.
func (r *DiskSubresource) Relocate(l object.VirtualDeviceList) (types.VirtualMachineRelocateSpecDiskLocator, error) {
	log.Printf("[DEBUG] %s: Starting relocate generation", r)
	device, err := r.FindVirtualDevice(l)
	var relocate types.VirtualMachineRelocateSpecDiskLocator
	if err != nil {
		return relocate, fmt.Errorf("cannot find disk device: %s", err)
	}
	disk, ok := device.(*types.VirtualDisk)
	if !ok {
		return relocate, fmt.Errorf("device at %q is not a virtual disk", l.Name(device))
	}

	// Expand all of the necessary disk settings first. This ensures all backing
	// data is properly populate and updated.
	if err := r.expandDiskSettings(disk); err != nil {
		return relocate, err
	}

	relocate.DiskId = disk.Key

	// Set the datastore for the relocation
	dsID := r.Get("datastore_id").(string)
	if dsID == "" {
		// Default to the default datastore
		dsID = r.rdd.Get("datastore_id").(string)
	}
	ds, err := datastore.FromID(r.client, dsID)
	if err != nil {
		return relocate, err
	}
	dsref := ds.Reference()
	relocate.Datastore = dsref

	// Add additional backing options if we are cloning.
	if r.rdd.Id() == "" {
		log.Printf("[DEBUG] %s: Adding additional options to relocator for cloning", r)

		// Set the new name. This is basically the same logic as create.
		diskName := r.Get("name").(string)
		vmxPath := r.rdd.Get("vmx_path").(string)
		if path.Base(diskName) == diskName && vmxPath != "" {
			diskName = path.Join(path.Dir(vmxPath), diskName)
		}

		backing := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo)
		backing.FileName = ds.Path(diskName)
		backing.Datastore = &dsref
		relocate.DiskBackingInfo = disk.Backing
	}

	// Done!
	log.Printf("[DEBUG] %s: Generated disk locator: %s", r, diskRelocateString(relocate))
	log.Printf("[DEBUG] %s: Relocate generation complete", r)
	return relocate, nil
}

// String prints out the disk sub-resource's information including the ID at
// time of instantiation, the short name of the disk, and the current device
// key and address.
func (r *DiskSubresource) String() string {
	n := r.Get("name")
	var name string
	if n != nil {
		name = path.Base(n.(string))
	} else {
		name = "<unknown>"
	}
	return fmt.Sprintf("%s (%s)", r.Subresource.String(), name)
}

// expandDiskSettings sets appropriate fields on an existing disk - this is
// used during Create and Update to set attributes to those found in
// configuration.
func (r *DiskSubresource) expandDiskSettings(disk *types.VirtualDisk) error {
	// Backing settings
	b := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo)
	b.DiskMode = r.GetWithRestart("disk_mode").(string)
	b.WriteThrough = structure.BoolPtr(r.GetWithRestart("write_through").(bool))

	// Only use disk_sharing if we are on vSphere 6.0 and higher
	version := viapi.ParseVersionFromClient(r.client)
	if version.Newer(viapi.VSphereVersion{Product: version.Product, Major: 6}) {
		b.Sharing = r.GetWithRestart("disk_sharing").(string)
	}

	// This settings are only set for internal disks
	if !r.Get("attach").(bool) {
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
	}

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
		dsID = r.rdd.Get("datastore_id").(string)
	}
	ds, err := datastore.FromID(r.client, dsID)
	if err != nil {
		return nil, err
	}
	dsref := ds.Reference()

	// Determine the full path to the disk if no directory is specified. The path
	// is the same path as the VMX file's current location. If we don't have a
	// VMX file path right now, don't worry about it - this means that the VM is
	// just being created and the file will be created in a directory of the same
	// name as the VMX file that is being created.
	diskName := r.Get("name").(string)
	vmxPath := r.rdd.Get("vmx_path").(string)
	if path.Base(diskName) == diskName && vmxPath != "" {
		diskName = path.Join(path.Dir(vmxPath), diskName)
	}

	disk := &types.VirtualDisk{
		VirtualDevice: types.VirtualDevice{
			Backing: &types.VirtualDiskFlatVer2BackingInfo{
				VirtualDeviceFileBackingInfo: types.VirtualDeviceFileBackingInfo{
					FileName:  ds.Path(diskName),
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
	sc, ok := ctlr.(types.BaseVirtualSCSIController)
	if !ok {
		return -1, nil, fmt.Errorf("controller at key %d is not a SCSI controller (actual: %T)", ctlr.GetVirtualDevice().Key, ctlr)
	}
	unit := *disk.UnitNumber
	if unit > sc.GetVirtualSCSIController().ScsiCtlrUnitNumber {
		unit--
	}
	unit = unit + 15*sc.GetVirtualSCSIController().BusNumber
	return int(unit), ctlr.(types.BaseVirtualController), nil
}

// diskRelocateListString pretty-prints a list of
// VirtualMachineRelocateSpecDiskLocator.
func diskRelocateListString(relocators []types.VirtualMachineRelocateSpecDiskLocator) string {
	var out []string
	for _, relocate := range relocators {
		out = append(out, diskRelocateString(relocate))
	}
	return strings.Join(out, ",")
}

// diskRelocateString prints out information from a
// VirtualMachineRelocateSpecDiskLocator in a friendly way.
//
// The format depends on whether or not a backing has been defined.
func diskRelocateString(relocate types.VirtualMachineRelocateSpecDiskLocator) string {
	key := relocate.DiskId
	var locstring string
	if backing, ok := relocate.DiskBackingInfo.(*types.VirtualDiskFlatVer2BackingInfo); ok && backing != nil {
		locstring = backing.FileName
	} else {
		locstring = relocate.Datastore.Value
	}
	return fmt.Sprintf("(%d => %s)", key, locstring)
}

// virtualDeviceListSorter is an internal type to facilitate sorting of a BaseVirtualDeviceList.
type virtualDeviceListSorter struct {
	Sort       object.VirtualDeviceList
	DeviceList object.VirtualDeviceList
}

// Len implements sort.Interface for virtualDeviceListSorter.
func (l virtualDeviceListSorter) Len() int {
	return len(l.Sort)
}

// Less helps implement sort.Interface for virtualDeviceListSorter. A
// BaseVirtualDevice is "less" than another device if its controller's bus
// number and unit number combination are earlier in the order than the other.
func (l virtualDeviceListSorter) Less(i, j int) bool {
	li := l.Sort[i]
	lj := l.Sort[j]
	liCtlr := l.DeviceList.FindByKey(li.GetVirtualDevice().ControllerKey)
	ljCtlr := l.DeviceList.FindByKey(lj.GetVirtualDevice().ControllerKey)
	if liCtlr == nil || ljCtlr == nil {
		panic(errors.New("virtualDeviceListSorter cannot be used with devices that are not assigned to a controller"))
	}
	if liCtlr.(types.BaseVirtualController).GetVirtualController().BusNumber < liCtlr.(types.BaseVirtualController).GetVirtualController().BusNumber {
		return true
	}
	liUnit := li.GetVirtualDevice().UnitNumber
	ljUnit := lj.GetVirtualDevice().UnitNumber
	if liUnit == nil || ljUnit == nil {
		panic(errors.New("virtualDeviceListSorter cannot be used with devices that do not have unit numbers set"))
	}
	return *liUnit < *ljUnit
}

// Swap helps implement sort.Interface for virtualDeviceListSorter.
func (l virtualDeviceListSorter) Swap(i, j int) {
	l.Sort[i], l.Sort[j] = l.Sort[j], l.Sort[i]
}

// virtualDiskSubresourceSorter sorts a list of disk sub-resources, based on unit number.
type virtualDiskSubresourceSorter []interface{}

// Len implements sort.Interface for virtualDiskSubresourceSorter.
func (s virtualDiskSubresourceSorter) Len() int {
	return len(s)
}

// Less helps implement sort.Interface for virtualDiskSubresourceSorter.
func (s virtualDiskSubresourceSorter) Less(i, j int) bool {
	mi := s[i].(map[string]interface{})
	mj := s[j].(map[string]interface{})
	return mi["unit_number"].(int) < mj["unit_number"].(int)
}

// Swap helps implement sort.Interface for virtualDiskSubresourceSorter.
func (s virtualDiskSubresourceSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// walkDiskBacking walks up a disk backing's parent disk chain looking for the
// supplied file name. It returns the full filename/datastore combination when
// it finds it, otherwise returns nothing.
func walkDiskBacking(b *types.VirtualDiskFlatVer2BackingInfo, name string) string {
	if datastorePathHasBase(b.FileName, name) {
		return b.FileName
	}
	if b.Parent != nil {
		return walkDiskBacking(b.Parent, name)
	}
	return ""
}

// datastorePathHasBase is a helper to check if a datastore path's file matches
// a supplied file name.
func datastorePathHasBase(p, b string) bool {
	dp := &object.DatastorePath{}
	if ok := dp.FromString(p); !ok {
		return false
	}
	return path.Base(dp.Path) == path.Base(b)
}

// selectDisks looks for disks that Terraform is supposed to manage. count is
// the number of controllers that Terraform is managing and serves as an upper
// limit (count - 1) of the SCSI bus number for a controller that eligible
// disks need to be attached to.
func selectDisks(l object.VirtualDeviceList, count int) object.VirtualDeviceList {
	devices := l.Select(func(device types.BaseVirtualDevice) bool {
		if disk, ok := device.(*types.VirtualDisk); ok {
			ctlr, err := findControllerForDevice(l, disk)
			if err != nil {
				log.Printf("[DEBUG] DiskRefreshOperation: Error looking for controller for device %q: %s", l.Name(disk), err)
				return false
			}
			if sc, ok := ctlr.(types.BaseVirtualSCSIController); ok && sc.GetVirtualSCSIController().BusNumber < int32(count) {
				cd := sc.(types.BaseVirtualDevice)
				log.Printf("[DEBUG] DiskRefreshOperation: Found controller %q for device %q", l.Name(cd), l.Name(disk))
				return true
			}
		}
		return false
	})
	return devices
}
