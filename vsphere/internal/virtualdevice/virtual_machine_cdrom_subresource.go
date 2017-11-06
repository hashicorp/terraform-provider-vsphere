package virtualdevice

import (
	"fmt"
	"log"
	"reflect"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/datastore"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

// CdromSubresourceSchema represents the schema for the cdrom sub-resource.
func CdromSubresourceSchema() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		// VirtualDeviceFileBackingInfo
		"datastore_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The datastore ID the ISO is located on.",
		},
		"path": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The path to the ISO file on the datastore.",
		},
	}
	structure.MergeSchema(s, subresourceSchema())
	return s
}

// CdromSubresource represents a vsphere_virtual_machine cdrom sub-resource,
// with a complex device lifecycle.
type CdromSubresource struct {
	*Subresource
}

// NewCdromSubresource returns a subresource populated with all of the necessary
// fields.
func NewCdromSubresource(client *govmomi.Client, rd *schema.ResourceData, d, old map[string]interface{}, idx int) *CdromSubresource {
	sr := &CdromSubresource{
		Subresource: &Subresource{
			schema:       CdromSubresourceSchema(),
			client:       client,
			srtype:       subresourceTypeCdrom,
			data:         d,
			olddata:      old,
			resourceData: rd,
		},
	}
	sr.Index = idx
	return sr
}

// CdromApplyOperation processes an apply operation for all disks in the
// resource.
//
// The function takes the root resource's ResourceData, the provider
// connection, and the device list as known to vSphere at the start of this
// operation. All disk operations are carried out, with both the complete,
// updated, VirtualDeviceList, and the complete list of changes returned as a
// slice of BaseVirtualDeviceConfigSpec.
func CdromApplyOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) (object.VirtualDeviceList, []types.BaseVirtualDeviceConfigSpec, error) {
	log.Printf("[DEBUG] CdromApplyOperation: Beginning apply operation")
	// While we are currently only restricting CD devices to one device, we have
	// to actually account for the fact that someone could add multiple CD drives
	// out of band. So this workflow is similar to the multi-device workflow that
	// exists for network devices.
	o, n := d.GetChange(subresourceTypeCdrom)
	ods := o.([]interface{})
	nds := n.([]interface{})

	var spec []types.BaseVirtualDeviceConfigSpec

	// Our old and new sets now have an accurate description of devices that may
	// have been added, removed, or changed. Look for removed devices first.
	log.Printf("[DEBUG] CdromApplyOperation: Looking for resources to delete")
nextOld:
	for n, oe := range ods {
		om := oe.(map[string]interface{})
		for _, ne := range nds {
			nm := ne.(map[string]interface{})
			if om["key"] == nm["key"] {
				continue nextOld
			}
		}
		r := NewCdromSubresource(c, d, om, nil, n)
		dspec, err := r.Delete(l)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		l = applyDeviceChange(l, dspec)
		spec = append(spec, dspec...)
	}

	// Now check for creates and updates. The results of this operation are
	// committed to state after the operation completes.
	var updates []interface{}
	log.Printf("[DEBUG] CdromApplyOperation: Looking for resources to create or update")
	for n, ne := range nds {
		nm := ne.(map[string]interface{})
		if n < len(ods) {
			// This is an update
			oe := ods[n]
			om := oe.(map[string]interface{})
			if nm["key"] != om["key"] {
				return nil, nil, fmt.Errorf("key mismatch on %s.%d (old: %d, new: %d). This is a bug with Terraform, please report it", subresourceTypeCdrom, n, nm["key"].(int), om["key"].(int))
			}
			if reflect.DeepEqual(nm, om) {
				// no change is a no-op
				updates = append(updates, nm)
				log.Printf("[DEBUG] CdromApplyOperation: No-op resource: key %d", nm["key"].(int))
				continue
			}
			r := NewCdromSubresource(c, d, nm, om, n)
			uspec, err := r.Update(l)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
			}
			l = applyDeviceChange(l, uspec)
			spec = append(spec, uspec...)
			updates = append(updates, r.Data())
			continue
		}
		// New device
		r := NewCdromSubresource(c, d, nm, nil, n)
		cspec, err := r.Create(l)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		l = applyDeviceChange(l, cspec)
		spec = append(spec, cspec...)
		updates = append(updates, r.Data())
	}

	log.Printf("[DEBUG] CdromApplyOperation: Post-apply final resource list: %s", subresourceListString(updates))
	// We are now done! Return the updated device list and config spec. Save updates as well.
	if err := d.Set(subresourceTypeCdrom, updates); err != nil {
		return nil, nil, err
	}
	log.Printf("[DEBUG] CdromApplyOperation: Device list at end of operation: %s", DeviceListString(l))
	log.Printf("[DEBUG] CdromApplyOperation: Device config operations from apply: %s", DeviceChangeString(spec))
	log.Printf("[DEBUG] CdromApplyOperation: Apply complete, returning updated spec")
	return l, spec, nil
}

// CdromRefreshOperation processes a refresh operation for all of the disks in
// the resource.
//
// This functions similar to CdromApplyOperation, but nothing to change is
// returned, all necessary values are just set and committed to state.
func CdromRefreshOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) error {
	log.Printf("[DEBUG] CdromRefreshOperation: Beginning refresh")
	// While we are currently only restricting CD devices to one device, we have
	// to actually account for the fact that someone could add multiple CD drives
	// out of band. So this workflow is similar to the multi-device workflow that
	// exists for network devices.
	devices := l.Select(func(device types.BaseVirtualDevice) bool {
		if _, ok := device.(*types.VirtualCdrom); ok {
			return true
		}
		return false
	})
	log.Printf("[DEBUG] CdromRefreshOperation: CDROM devices located: %s", DeviceListString(devices))
	curSet := d.Get(subresourceTypeCdrom).([]interface{})
	log.Printf("[DEBUG] CdromRefreshOperation: Current resource set from state: %s", subresourceListString(curSet))
	var newSet []interface{}
	// First check for negative keys. These are freshly added devices that are
	// usually coming into read post-create.
	//
	// If we find what we are looking for, we remove the device from the working
	// set so that we don't try and process it in the next few passes.
	log.Printf("[DEBUG] CdromRefreshOperation: Looking for freshly-created resources to read in")
	for n, item := range curSet {
		m := item.(map[string]interface{})
		if m["key"].(int) < 1 {
			r := NewCdromSubresource(c, d, m, nil, n)
			if err := r.Read(l); err != nil {
				return fmt.Errorf("%s: %s", r.Addr(), err)
			}
			newM := r.Data()
			if newM["key"].(int) < 1 {
				// This should not have happened - if it did, our device
				// creation/update logic failed somehow that we were not able to track.
				return fmt.Errorf("device %d with address %s still unaccounted for after update/read", newM["key"].(int), newM["device_address"].(string))
			}
			newSet = append(newSet, r.Data())
			for i := 0; i < len(devices); i++ {
				device := devices[i]
				if device.GetVirtualDevice().Key == int32(newM["key"].(int)) {
					devices = append(devices[:i], devices[i+1:]...)
					i--
				}
			}
		}
	}
	log.Printf("[DEBUG] CdromRefreshOperation: CDROM devices after freshly-created device search: %s", DeviceListString(devices))
	log.Printf("[DEBUG] CdromRefreshOperation: Resource set to write after freshly-created device search: %s", subresourceListString(newSet))

	// Go over the remaining devices, refresh via key, and then remove their
	// entries as well.
	log.Printf("[DEBUG] CdromRefreshOperation: Looking for devices known in state")
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
			r := NewCdromSubresource(c, d, m, nil, n)
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
	log.Printf("[DEBUG] CdromRefreshOperation: Resource set to write after known device search: %s", subresourceListString(newSet))
	log.Printf("[DEBUG] CdromRefreshOperation: Probable orphaned CDROM devices: %s", DeviceListString(devices))

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
		r := NewCdromSubresource(c, d, m, nil, n)
		if err := r.Read(l); err != nil {
			return fmt.Errorf("%s: %s", r.Addr(), err)
		}
		newSet = append(newSet, r.Data())
	}

	log.Printf("[DEBUG] CdromRefreshOperation: Resource set to write after adding orphaned devices: %s", subresourceListString(newSet))
	log.Printf("[DEBUG] CdromRefreshOperation: Refresh operation complete, sending new resource set")
	return d.Set(subresourceTypeCdrom, newSet)
}

// Create creates a vsphere_virtual_machine cdrom sub-resource.
func (r *CdromSubresource) Create(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	log.Printf("[DEBUG] %s: Running create", r)
	var spec []types.BaseVirtualDeviceConfigSpec
	var ctlr types.BaseVirtualController
	ctlr, err := r.ControllerForCreateUpdate(l, SubresourceControllerTypeIDE, 0)
	if err != nil {
		return nil, err
	}

	// We now have the controller on which we can create our device on.
	dsID := r.Get("datastore_id").(string)
	path := r.Get("path").(string)
	ds, err := datastore.FromID(r.client, dsID)
	if err != nil {
		return nil, fmt.Errorf("cannot find datastore: %s", err)
	}
	dsProps, err := datastore.Properties(ds)
	if err != nil {
		return nil, fmt.Errorf("could not get properties for datastore: %s", err)
	}
	dsName := dsProps.Name

	dsPath := &object.DatastorePath{
		Datastore: dsName,
		Path:      path,
	}

	device, err := l.CreateCdrom(ctlr.(*types.VirtualIDEController))
	if err != nil {
		return nil, err
	}
	device = l.InsertIso(device, dsPath.String())

	// Done here. Save IDs, push the device to the new device list and return.
	r.SaveDevIDs(device, ctlr)
	dspec, err := object.VirtualDeviceList{device}.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
	if err != nil {
		return nil, err
	}
	spec = append(spec, dspec...)
	log.Printf("[DEBUG] %s: Device config operations from create: %s", r, DeviceChangeString(spec))
	log.Printf("[DEBUG] %s: Create finished", r)
	return spec, nil
}

// Read reads a vsphere_virtual_machine cdrom sub-resource.
func (r *CdromSubresource) Read(l object.VirtualDeviceList) error {
	log.Printf("[DEBUG] %s: Reading state", r)
	d, err := r.FindVirtualDevice(l)
	if err != nil {
		return fmt.Errorf("cannot find disk device: %s", err)
	}
	device, ok := d.(*types.VirtualCdrom)
	if !ok {
		return fmt.Errorf("device at %q is not a virtual CDROM device", l.Name(d))
	}
	backing := device.Backing.(*types.VirtualCdromIsoBackingInfo)
	dp := &object.DatastorePath{}
	if ok := dp.FromString(backing.FileName); !ok {
		return fmt.Errorf("could not read datastore path in backing %q", backing.FileName)
	}
	r.Set("datastore_id", backing.Datastore.Value)
	r.Set("path", dp.Path)
	// Save the device key and address data
	ctlr, err := findControllerForDevice(l, d)
	if err != nil {
		return err
	}
	r.SaveDevIDs(d, ctlr)
	log.Printf("[DEBUG] %s: Read finished (key and device address may have changed)", r)
	return nil
}

// Update updates a vsphere_virtual_machine cdrom sub-resource.
func (r *CdromSubresource) Update(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	log.Printf("[DEBUG] %s: Beginning update", r)
	d, err := r.FindVirtualDevice(l)
	if err != nil {
		return nil, fmt.Errorf("cannot find disk device: %s", err)
	}
	device, ok := d.(*types.VirtualCdrom)
	if !ok {
		return nil, fmt.Errorf("device at %q is not a virtual CDROM device", l.Name(d))
	}

	// To update, we just re-insert the ISO as per create, and send it as an edit.
	dsID := r.Get("datastore_id").(string)
	path := r.Get("path").(string)
	ds, err := datastore.FromID(r.client, dsID)
	if err != nil {
		return nil, fmt.Errorf("cannot find datastore: %s", err)
	}
	dsProps, err := datastore.Properties(ds)
	if err != nil {
		return nil, fmt.Errorf("could not get properties for datastore: %s", err)
	}
	dsName := dsProps.Name

	dsPath := &object.DatastorePath{
		Datastore: dsName,
		Path:      path,
	}

	device = l.InsertIso(device, dsPath.String())

	spec, err := object.VirtualDeviceList{device}.ConfigSpec(types.VirtualDeviceConfigSpecOperationEdit)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] %s: Device config operations from update: %s", r, DeviceChangeString(spec))
	log.Printf("[DEBUG] %s: Update complete", r)
	return spec, nil
}

// Delete deletes a vsphere_virtual_machine cdrom sub-resource.
func (r *CdromSubresource) Delete(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	log.Printf("[DEBUG] %s: Beginning delete", r)
	d, err := r.FindVirtualDevice(l)
	if err != nil {
		return nil, fmt.Errorf("cannot find disk device: %s", err)
	}
	device, ok := d.(*types.VirtualCdrom)
	if !ok {
		return nil, fmt.Errorf("device at %q is not a virtual CDROM device", l.Name(d))
	}
	deleteSpec, err := object.VirtualDeviceList{device}.ConfigSpec(types.VirtualDeviceConfigSpecOperationRemove)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] %s: Device config operations from update: %s", r, DeviceChangeString(deleteSpec))
	log.Printf("[DEBUG] %s: Delete completed", r)
	return deleteSpec, nil
}
