package vsphere

import (
	"context"
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// rootPathParticle is the section of a vSphere inventory path that denotes a
// specific kind of inventory item.
type rootPathParticle string

// String implements Stringer for rootPathParticle.
func (p rootPathParticle) String() string {
	return string(p)
}

// Delimeter returns the path delimiter for the particle, which is basically
// just a particle with a leading slash.
func (p rootPathParticle) Delimeter() string {
	return string("/" + p)
}

// SplitDatacenter is a convenience method that splits out the datacenter path
// from the supplied path for the particle.
func (p rootPathParticle) SplitDatacenter(inventoryPath string) (string, error) {
	s := strings.SplitN(inventoryPath, p.Delimeter(), 2)
	if len(s) != 2 {
		return inventoryPath, fmt.Errorf("could not split path %q on %q", inventoryPath, p.Delimeter())
	}
	return s[0], nil
}

// SplitRelativeFolder is a convenience method that splits out the relative
// folder from the supplied path for the particle.
func (p rootPathParticle) SplitRelativeFolder(inventoryPath string) (string, error) {
	s := strings.SplitN(inventoryPath, p.Delimeter(), 2)
	if len(s) != 2 {
		return inventoryPath, fmt.Errorf("could not split path %q on %q", inventoryPath, p.Delimeter())
	}
	return path.Dir(s[1]), nil
}

// NewRootFromPath takes the datacenter path for a specific entity, and then
// appends the new particle supplied.
func (p rootPathParticle) NewRootFromPath(inventoryPath string, newParticle rootPathParticle) (string, error) {
	dcPath, err := p.SplitDatacenter(inventoryPath)
	if err != nil {
		return inventoryPath, err
	}
	return fmt.Sprintf("%s/%s", dcPath, newParticle), nil
}

// PathFromNewRoot takes the datacenter path for a specific entity, and then
// appends the new particle supplied with the new relative path.
//
// As an example, consider a supplied host path "/dc1/host/cluster1/esxi1", and
// a supplied datastore folder relative path of "/foo/bar".  This function will
// split off the datacenter section of the path (/dc1) and combine it with the
// datastore folder with the proper delimiter. The resulting path will be
// "/dc1/datastore/foo/bar".
func (p rootPathParticle) PathFromNewRoot(inventoryPath string, newParticle rootPathParticle, relative string) (string, error) {
	rootPath, err := p.NewRootFromPath(inventoryPath, newParticle)
	if err != nil {
		return inventoryPath, err
	}
	return path.Clean(fmt.Sprintf("%s/%s", rootPath, relative)), nil
}

const (
	rootPathParticleVM        = rootPathParticle("vm")
	rootPathParticleNetwork   = rootPathParticle("network")
	rootPathParticleHost      = rootPathParticle("host")
	rootPathParticleDatastore = rootPathParticle("datastore")
)

// datacenterPathFromHostSystemID returns the datacenter section of a
// HostSystem's inventory path.
func datacenterPathFromHostSystemID(client *govmomi.Client, hsID string) (string, error) {
	hs, err := hostSystemFromID(client, hsID)
	if err != nil {
		return "", err
	}
	return rootPathParticleHost.SplitDatacenter(hs.InventoryPath)
}

// datastoreRootPathFromHostSystemID returns the root datastore folder path
// for a specific host system ID.
func datastoreRootPathFromHostSystemID(client *govmomi.Client, hsID string) (string, error) {
	hs, err := hostSystemFromID(client, hsID)
	if err != nil {
		return "", err
	}
	return rootPathParticleHost.NewRootFromPath(hs.InventoryPath, rootPathParticleDatastore)
}

// folderFromObject returns an *object.Folder from a given object of specific
// types, and relative path of a type defined in folderType. If no such folder
// is found, an appropriate error will be returned.
//
// The list of supported object types will grow as the provider supports more
// resources.
func folderFromObject(client *govmomi.Client, obj interface{}, folderType rootPathParticle, relative string) (*object.Folder, error) {
	if err := validateVirtualCenter(client); err != nil {
		return nil, err
	}
	var p string
	var err error
	switch o := obj.(type) {
	case (*object.Datastore):
		p, err = rootPathParticleDatastore.PathFromNewRoot(o.InventoryPath, folderType, relative)
	case (*object.HostSystem):
		p, err = rootPathParticleHost.PathFromNewRoot(o.InventoryPath, folderType, relative)
	default:
		return nil, fmt.Errorf("unsupported object type %T", o)
	}
	if err != nil {
		return nil, err
	}
	// Set up a finder. Don't set datacenter here as we are looking for full
	// path, should not be necessary.
	finder := find.NewFinder(client.Client, false)
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	folder, err := finder.Folder(ctx, p)
	if err != nil {
		return nil, err
	}
	return folder, nil
}

// datastoreFolderFromObject returns an *object.Folder from a given object,
// and relative datastore folder path. If no such folder is found, of if it is
// not a datastore folder, an appropriate error will be returned.
func datastoreFolderFromObject(client *govmomi.Client, obj interface{}, relative string) (*object.Folder, error) {
	folder, err := folderFromObject(client, obj, rootPathParticleDatastore, relative)
	if err != nil {
		return nil, err
	}

	return validateDatastoreFolder(folder)
}

// validateDatastoreFolder checks to make sure the folder is a datastore
// folder, and returns it if it is not, or an error if it isn't.
func validateDatastoreFolder(folder *object.Folder) (*object.Folder, error) {
	var props mo.Folder
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	if err := folder.Properties(ctx, folder.Reference(), nil, &props); err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(props.ChildType, []string{"Folder", "Datastore", "StoragePod"}) {
		return nil, fmt.Errorf("%q is not a datastore folder", folder.InventoryPath)
	}
	return folder, nil
}

// pathIsEmpty checks a folder path to see if it's "empty" (ie: would resolve
// to the root inventory path for a given type in a datacenter - "" or "/").
func pathIsEmpty(path string) bool {
	return path == "" || path == "/"
}

// normalizeFolderPath is a SchemaStateFunc that normalizes a folder path.
func normalizeFolderPath(v interface{}) string {
	p := v.(string)
	if pathIsEmpty(p) {
		return ""
	}
	return strings.TrimPrefix(path.Clean(p), "/")
}

// moveObjectToFolder moves a object by reference into a folder.
func moveObjectToFolder(ref types.ManagedObjectReference, folder *object.Folder) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	task, err := folder.MoveInto(ctx, []types.ManagedObjectReference{ref})
	if err != nil {
		return err
	}
	tctx, tcancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer tcancel()
	return task.Wait(tctx)
}
