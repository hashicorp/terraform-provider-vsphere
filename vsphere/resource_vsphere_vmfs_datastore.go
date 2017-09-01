package vsphere

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/types"
)

const (
	retryDeletePending   = "retryDeletePending"
	retryDeleteCompleted = "retryDeleteCompleted"
	retryDeleteError     = "retryDeleteError"

	waitForDeletePending   = "waitForDeletePending"
	waitForDeleteCompleted = "waitForDeleteCompleted"
	waitForDeleteError     = "waitForDeleteError"
)

// formatCreateRollbackError defines the verbose error for extending a disk on
// creation where rollback is not possible.
const formatCreateRollbackErrorUpdate = `
WARNING: Dangling resource!
There was an error extending your datastore with disk: %q:
%s
Additionally, there was an error removing the created datastore:
%s
You will need to remove this datastore manually before trying again.
`

// formatCreateRollbackError defines the verbose error for extending a disk on
// creation where rollback is not possible.
const formatCreateRollbackErrorProperties = `
WARNING: Dangling resource!
After creating the datastore, there was an error fetching its properties:
%s
Additionally, there was an error removing the created datastore:
%s
You will need to remove this datastore manually before trying again.
`

// formatUpdateInconsistentState defines the verbose error when the setting
// state failed in the middle of an update opeartion. This is an error that
// will require repair of the state before TF can continue.
const formatUpdateInconsistentState = `
WARNING: Inconsistent state!
Terraform was able to add disk %q, but could not save the update to the state.
The error was:
%s
This is more than likely a bug. Please report it at:
https://github.com/terraform-providers/terraform-provider-vsphere/issues
Also, please try running "terraform refresh" to try to update the state before
trying again. If this fails, you may need to repair the state manually.
`

func resourceVSphereVmfsDatastore() *schema.Resource {
	s := map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:        schema.TypeString,
			Description: "The name of the datastore.",
			Required:    true,
		},
		"host_system_id": &schema.Schema{
			Type:        schema.TypeString,
			Description: "The managed object ID of the host to set up the datastore on.",
			Required:    true,
		},
		"disks": &schema.Schema{
			Type:        schema.TypeList,
			Description: "The disks to add to the datastore.",
			Required:    true,
			MinItems:    1,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
	}
	mergeSchema(s, schemaDatastoreSummary())
	return &schema.Resource{
		Create: resourceVSphereVmfsDatastoreCreate,
		Read:   resourceVSphereVmfsDatastoreRead,
		Update: resourceVSphereVmfsDatastoreUpdate,
		Delete: resourceVSphereVmfsDatastoreDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVSphereVmfsDatastoreImport,
		},
		Schema: s,
	}
}

func resourceVSphereVmfsDatastoreCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	hsID := d.Get("host_system_id").(string)
	dss, err := hostDatastoreSystemFromHostSystemID(client, hsID)
	if err != nil {
		return fmt.Errorf("error loading host datastore system: %s", err)
	}

	// To ensure the datastore is fully created with all the disks that we want
	// to add to it, first we add the initial disk, then we expand the disk with
	// the rest of the extents.
	disks := d.Get("disks").([]interface{})
	disk := disks[0].(string)
	spec, err := diskSpecForCreate(dss, disk)
	if err != nil {
		return err
	}
	spec.Vmfs.VolumeName = d.Get("name").(string)
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	ds, err := dss.CreateVmfsDatastore(ctx, *spec)
	if err != nil {
		return fmt.Errorf("error creating datastore with disk %s: %s", disk, err)
	}

	// Now add any remaining disks.
	for _, disk := range disks[1:] {
		spec, err := diskSpecForExtend(dss, ds, disk.(string))
		if err != nil {
			// We have to destroy the created datastore here.
			if remErr := removeDatastore(dss, ds); remErr != nil {
				// We could not destroy the created datastore and there is now a dangling
				// resource. We need to instruct the user to remove the datastore
				// manually.
				return fmt.Errorf(formatCreateRollbackErrorUpdate, disk, err, remErr)
			}
			return fmt.Errorf("error fetching datastore extend spec for disk %q: %s", disk, err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
		defer cancel()
		if _, err := extendVmfsDatastore(ctx, dss, ds, *spec); err != nil {
			if remErr := removeDatastore(dss, ds); remErr != nil {
				// We could not destroy the created datastore and there is now a dangling
				// resource. We need to instruct the user to remove the datastore
				// manually.
				return fmt.Errorf(formatCreateRollbackErrorUpdate, disk, err, remErr)
			}
			return fmt.Errorf("error extending datastore with disk %q: %s", disk, err)
		}
	}

	d.SetId(ds.Reference().Value)

	// Done
	return resourceVSphereVmfsDatastoreRead(d, meta)
}

func resourceVSphereVmfsDatastoreRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	id := d.Id()
	ds, err := datastoreFromID(client, id)
	if err != nil {
		return fmt.Errorf("cannot find datastore: %s", err)
	}
	props, err := datastoreProperties(ds)
	if err != nil {
		return fmt.Errorf("could not get properties for datastore: %s", err)
	}
	if err := flattenDatastoreSummary(d, &props.Summary); err != nil {
		return err
	}

	// We also need to update the disk list from the summary.
	var disks []string
	for _, disk := range props.Info.(*types.VmfsDatastoreInfo).Vmfs.Extent {
		disks = append(disks, disk.DiskName)
	}
	if err := d.Set("disks", disks); err != nil {
		return err
	}
	return nil
}

func resourceVSphereVmfsDatastoreUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	hsID := d.Get("host_system_id").(string)
	dss, err := hostDatastoreSystemFromHostSystemID(client, hsID)
	if err != nil {
		return fmt.Errorf("error loading host datastore system: %s", err)
	}

	id := d.Id()
	ds, err := datastoreFromID(client, id)
	if err != nil {
		return fmt.Errorf("cannot find datastore: %s", err)
	}

	// Rename this datastore if our name has drifted.
	if d.HasChange("name") {
		if err := renameObject(client, ds.Reference(), d.Get("name").(string)); err != nil {
			return err
		}
	}

	// Veto this update if it means a disk was removed. Shrinking
	// datastores/removing extents is not supported.
	old, new := d.GetChange("disks")
	for _, v1 := range old.([]interface{}) {
		var found bool
		for _, v2 := range new.([]interface{}) {
			if v1.(string) == v2.(string) {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("disk %s found in state but not config (removal of disks is not supported)", v1)
		}
	}

	// Maintain a copy of the disks that have been added just in case the update
	// fails in the middle.
	old2 := make([]interface{}, len(old.([]interface{})))
	copy(old2, old.([]interface{}))

	// Now we basically reverse what we did above when we were checking for
	// removed disks, and add any new disks that have been added.
	for _, v1 := range new.([]interface{}) {
		var found bool
		for _, v2 := range old.([]interface{}) {
			if v1.(string) == v2.(string) {
				found = true
			}
		}
		if !found {
			// Add the disk
			spec, err := diskSpecForExtend(dss, ds, v1.(string))
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
			defer cancel()
			if _, err := extendVmfsDatastore(ctx, dss, ds, *spec); err != nil {
				return err
			}
			// The add was successful. Since this update is not atomic, we need to at
			// least update the disk list to make sure we don't attempt to add the
			// same disk twice.
			old2 = append(old2, v1)
			if err := d.Set("disks", old2); err != nil {
				return fmt.Errorf(formatUpdateInconsistentState, v1.(string), err)
			}
		}
	}

	// Should be done with the update here.
	return resourceVSphereVmfsDatastoreRead(d, meta)
}

func resourceVSphereVmfsDatastoreDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	hsID := d.Get("host_system_id").(string)
	dss, err := hostDatastoreSystemFromHostSystemID(client, hsID)
	if err != nil {
		return fmt.Errorf("error loading host datastore system: %s", err)
	}

	id := d.Id()
	ds, err := datastoreFromID(client, id)
	if err != nil {
		return fmt.Errorf("cannot find datastore: %s", err)
	}

	// This is a race that more than likely will only come up during tests, but
	// we still want to guard against it - when working with datastores that end
	// up mounting across multiple hosts, removing the datastore will fail if
	// it's removed too quickly (like right away, for example). So we set up a
	// very short retry waiter to make sure if the first attempt fails, the
	// second one should probably succeed right away. We also insert a small
	// minimum delay to make an honest first attempt at trying to delete the
	// datastore without spamming the task log with errors.
	deleteRetryFunc := func() (interface{}, string, error) {
		err := removeDatastore(dss, ds)
		if err != nil {
			if isResourceInUseError(err) {
				// Pending
				return struct{}{}, retryDeletePending, nil
			}
			// Some other error
			return struct{}{}, retryDeleteError, err
		}
		// Done
		return struct{}{}, retryDeleteCompleted, nil
	}

	deleteRetry := &resource.StateChangeConf{
		Pending:    []string{retryDeletePending},
		Target:     []string{retryDeleteCompleted},
		Refresh:    deleteRetryFunc,
		Timeout:    30 * time.Second,
		MinTimeout: 2 * time.Second,
		Delay:      2 * time.Second,
	}

	_, err = deleteRetry.WaitForState()
	if err != nil {
		return fmt.Errorf("could not delete datastore: %s", err)
	}

	// We need to make sure the datastore is completely removed. There appears to
	// be a bit of a delay sometimes on vCenter, and it causes issues in tests,
	// which means it could cause issues somewhere else too.
	waitForDeleteFunc := func() (interface{}, string, error) {
		_, err := datastoreFromID(client, id)
		if err != nil {
			if isManagedObjectNotFoundError(err) {
				// Done
				return struct{}{}, waitForDeleteCompleted, nil
			}
			// Some other error
			return struct{}{}, waitForDeleteError, err
		}
		return struct{}{}, waitForDeletePending, nil
	}

	waitForDelete := &resource.StateChangeConf{
		Pending:        []string{waitForDeletePending},
		Target:         []string{waitForDeleteCompleted},
		Refresh:        waitForDeleteFunc,
		Timeout:        defaultAPITimeout,
		MinTimeout:     2 * time.Second,
		Delay:          1 * time.Second,
		NotFoundChecks: 35,
	}

	_, err = waitForDelete.WaitForState()
	if err != nil {
		return fmt.Errorf("error waiting for datastore to delete: %s", err.Error())
	}

	return nil
}

func resourceVSphereVmfsDatastoreImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// We support importing a MoRef - so we need to load the datastore and check
	// to make sure 1) it exists, and 2) it's a VMFS datastore. If it is, we are
	// good to go (rest of the stuff will be handled by read on refresh).
	client := meta.(*govmomi.Client)
	id := d.Id()
	ds, err := datastoreFromID(client, id)
	if err != nil {
		return nil, fmt.Errorf("cannot find datastore: %s", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	t, err := ds.Type(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching datastore type: %s", err)
	}
	if t != types.HostFileSystemVolumeFileSystemTypeVMFS {
		return nil, fmt.Errorf("datastore ID %q is not a VMFS datastore", id)
	}
	return []*schema.ResourceData{d}, nil
}
