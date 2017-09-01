package vsphere

import (
	"context"
	"fmt"

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
		return nil, fmt.Errorf("could not find datastore with id: %s: %s", id, err)
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

// datastoreIsMissing checks if the error messages returned from
// datastoreFromID indicates that the datastore is missing. This is used in
// various deletion checks.
func datastoreIsMissing(id string, err error) bool {
	msg := fmt.Sprintf("could not find datastore with id: %s: ServerFaultCode: The object 'vim.Datastore:%s' has already been deleted or has not been completely created", id, id)

	if err.Error() == msg {
		return true
	}
	return false
}
