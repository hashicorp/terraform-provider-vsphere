package virtualdevice

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/datastore"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

// cdromSubresourceSchema represents the schema for the cdrom sub-resource.
func cdromSubresourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
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
}

// CdromSubresource represents a vsphere_virtual_machine cdrom sub-resource,
// with a complex device lifecycle.
type CdromSubresource struct {
	*Subresource
}

// NewCdromSubresource returns a subresource populated with all of the necessary
// fields.
func NewCdromSubresource(client *govmomi.Client, index, oldindex int, d *schema.ResourceData) SubresourceInstance {
	sr := &CdromSubresource{
		Subresource: &Subresource{
			schema:   cdromSubresourceSchema(),
			client:   client,
			srtype:   subresourceTypeCdrom,
			index:    index,
			oldindex: oldindex,
			data:     d,
		},
	}
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
	return deviceApplyOperation(d, c, l, subresourceTypeCdrom, NewCdromSubresource)
}

// CdromRefreshOperation processes a refresh operation for all of the disks in
// the resource.
//
// This functions similar to CdromApplyOperation, but nothing to change is
// returned, all necessary values are just set and committed to state.
func CdromRefreshOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) error {
	return deviceRefreshOperation(d, c, l, subresourceTypeCdrom, NewCdromSubresource)
}

// Create creates a vsphere_virtual_machine cdrom sub-resource.
func (r *CdromSubresource) Create(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
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
	return spec, nil
}

// Read reads a vsphere_virtual_machine cdrom sub-resource and commits the data
// to the newData layer.
func (r *CdromSubresource) Read(l object.VirtualDeviceList) error {
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
	return nil
}

// Update updates a vsphere_virtual_machine disk sub-resource.
func (r *CdromSubresource) Update(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
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

	// Done here. Save ID, push the device to the new device list and return.
	spec, err := object.VirtualDeviceList{device}.ConfigSpec(types.VirtualDeviceConfigSpecOperationEdit)
	if err != nil {
		return nil, err
	}
	return spec, nil
}

// Delete deletes a vsphere_virtual_machine disk sub-resource.
func (r *CdromSubresource) Delete(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
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
	return deleteSpec, nil
}
