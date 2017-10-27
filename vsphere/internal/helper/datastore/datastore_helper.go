package datastore

import (
	"context"

	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/folder"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/hostsystem"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// FromID locates a Datastore by its managed object reference ID.
func FromID(client *govmomi.Client, id string) (*object.Datastore, error) {
	finder := find.NewFinder(client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  "Datastore",
		Value: id,
	}

	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
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

// Properties is a convenience method that wraps fetching the
// Datastore MO from its higher-level object.
func Properties(ds *object.Datastore) (*mo.Datastore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	var props mo.Datastore
	if err := ds.Properties(ctx, ds.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

// MoveToFolder is a complex method that moves a datastore to a given
// relative datastore folder path. "Relative" here means relative to a
// datacenter, which is discovered from the current datastore path.
func MoveToFolder(client *govmomi.Client, ds *object.Datastore, relative string) error {
	f, err := folder.DatastoreFolderFromObject(client, ds, relative)
	if err != nil {
		return err
	}
	return folder.MoveObjectTo(ds.Reference(), f)
}

// MoveToFolderRelativeHostSystemID is a complex method that moves a
// datastore to a given datastore path, similar to MoveToFolder,
// except the path is relative to a HostSystem supplied by ID instead of the
// datastore.
func MoveToFolderRelativeHostSystemID(client *govmomi.Client, ds *object.Datastore, hsID, relative string) error {
	hs, err := hostsystem.FromID(client, hsID)
	if err != nil {
		return err
	}
	f, err := folder.DatastoreFolderFromObject(client, hs, relative)
	if err != nil {
		return err
	}
	return folder.MoveObjectTo(ds.Reference(), f)
}
