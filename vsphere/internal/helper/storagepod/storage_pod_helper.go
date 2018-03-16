package storagepod

import (
	"context"
	"fmt"
	"log"

	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// FromID locates a StoragePod by its managed object reference ID.
func FromID(client *govmomi.Client, id string) (*object.StoragePod, error) {
	log.Printf("[DEBUG] Locating datastore cluster with ID %q", id)
	finder := find.NewFinder(client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  "StoragePod",
		Value: id,
	}

	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	r, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		return nil, err
	}
	pod := r.(*object.StoragePod)
	log.Printf("[DEBUG] Datastore cluster with ID %q found (%s)", pod.Reference().Value, pod.InventoryPath)
	return pod, nil
}

// FromPath loads a StoragePod from its path. The datacenter is optional if the
// path is specific enough to not require it.
func FromPath(client *govmomi.Client, name string, dc *object.Datacenter) (*object.StoragePod, error) {
	finder := find.NewFinder(client.Client, false)
	if dc != nil {
		log.Printf("[DEBUG] Attempting to locate datastore cluster %q in datacenter %q", name, dc.InventoryPath)
		finder.SetDatacenter(dc)
	} else {
		log.Printf("[DEBUG] Attempting to locate datastore cluster at absolute path %q", name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	return finder.DatastoreCluster(ctx, name)
}

// Properties is a convenience method that wraps fetching the
// StoragePod MO from its higher-level object.
func Properties(pod *object.StoragePod) (*mo.StoragePod, error) {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	var props mo.StoragePod
	if err := pod.Properties(ctx, pod.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

// Create creates a StoragePod from a supplied folder. The resulting StoragePod
// is returned.
func Create(f *object.Folder, name string) (*object.StoragePod, error) {
	log.Printf("[DEBUG] Creating datastore cluster %q", fmt.Sprintf("%s/%s", f.InventoryPath, name))
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	pod, err := f.CreateStoragePod(ctx, name)
	if err != nil {
		return nil, err
	}
	return pod, nil
}

// ApplyDRSConfiguration takes a types.StorageDrsConfigSpec and applies it
// against the specified StoragePod.
func ApplyDRSConfiguration(client *govmomi.Client, pod *object.StoragePod, spec types.StorageDrsConfigSpec) error {
	log.Printf("[DEBUG] Applying storage DRS configuration against datastore clsuter %q", pod.InventoryPath)
	mgr := object.NewStorageResourceManager(client.Client)
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	task, err := mgr.ConfigureStorageDrsForPod(ctx, pod, spec, true)
	if err != nil {
		return err
	}
	return task.Wait(ctx)
}
