package vsphere

import (
	"context"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// datastoreFromID locates a Datastore by its managed object reference ID.
func datastoreFromID(client *govmomi.Client, id string) (*object.Datastore, error) {
	finder := find.NewFinder(client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  "Datastore",
		Value: id,
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	ds, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		return nil, err
	}
	// Should be safe to return here. If our reference returned here and is not a
	// datastore, then we have bigger problems and to be honest we should be
	// panicking anyway.
	return ds.(*object.Datastore), nil
}

// datastoreProperties is a convenience method that wraps fetching the
// Datastore MO from its higher-level object.
func datastoreProperties(ds *object.Datastore) (*mo.Datastore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	var props mo.Datastore
	if err := ds.Properties(ctx, ds.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

// moveDatastoreToFolder is a complex method that moves a datastore to a given
// relative datastore folder path. "Relative" here means relative to a
// datacenter, which is discovered from the current datastore path.
func moveDatastoreToFolder(client *govmomi.Client, ds *object.Datastore, relative string) error {
	folder, err := datastoreFolderFromObject(client, ds, relative)
	if err != nil {
		return err
	}
	return moveObjectToFolder(ds.Reference(), folder)
}

// moveDatastoreToFolderRelativeHostSystemID is a complex method that moves a
// datastore to a given datastore path, similar to moveDatastoreToFolder,
// except the path is relative to a HostSystem supplied by ID instead of the
// datastore.
func moveDatastoreToFolderRelativeHostSystemID(client *govmomi.Client, ds *object.Datastore, hsID, relative string) error {
	hs, err := hostSystemFromID(client, hsID)
	if err != nil {
		return err
	}
	folder, err := datastoreFolderFromObject(client, hs, relative)
	if err != nil {
		return err
	}
	return moveObjectToFolder(ds.Reference(), folder)
}
