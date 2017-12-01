package computeresource

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/envbrowse"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// BaseComputeResource is an interface that ComputeResource and any derivative
// types will implement on part of having all of the methods available to
// ComputeResource. It also contains the Properties method from the
// common-level object method set.
//
// Its use is mainly to facilitate common functionality between the two in
// helpers.
type BaseComputeResource interface {
	Datastores(context.Context) ([]*object.Datastore, error)
	Destroy(context.Context) (*object.Task, error)
	Hosts(context.Context) ([]*object.HostSystem, error)
	Reconfigure(context.Context, types.BaseComputeResourceConfigSpec, bool) (*object.Task, error)
	ResourcePool(context.Context) (*object.ResourcePool, error)

	Name() string
	Properties(context.Context, types.ManagedObjectReference, []string, interface{}) error
	Reference() types.ManagedObjectReference
}

// StandaloneFromID locates a ComputeResource by its managed object reference ID.
//
// Note this is for base level ComputeResource objects only, and should only be
// used for standalone hosts. If you are looking for a cluster, use
// ClusterFromID.
func StandaloneFromID(client *govmomi.Client, id string) (*object.ComputeResource, error) {
	finder := find.NewFinder(client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  "ComputeResource",
		Value: id,
	}

	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	obj, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		return nil, err
	}
	return obj.(*object.ComputeResource), nil
}

// ClusterFromID returns a ClusterComputeResource, a subclass of
// ComputeResource that is used for clusters.
func ClusterFromID(client *govmomi.Client, id string) (*object.ClusterComputeResource, error) {
	finder := find.NewFinder(client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  "ClusterComputeResource",
		Value: id,
	}

	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	obj, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		return nil, err
	}
	return obj.(*object.ClusterComputeResource), nil
}

// BaseFromPath returns a BaseComputeResource for a given path.
func BaseFromPath(client *govmomi.Client, path string) (BaseComputeResource, error) {
	finder := find.NewFinder(client.Client, false)

	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	list, err := finder.ManagedObjectList(ctx, path, "ComputeResource", "ClusterComputeResource")
	if err != nil {
		return nil, err
	}
	if len(list) < 1 {
		return nil, fmt.Errorf("no compute resources found at path %q", path)
	}
	if len(list) > 1 {
		return nil, fmt.Errorf("multiple results returned for path %q", path)
	}
	if !strings.HasSuffix(list[0].Path, path) {
		return nil, fmt.Errorf("returned object path %q does not properly match search path %q", list[0].Path, path)
	}
	return BaseFromReference(client, list[0].Object.Reference())
}

// BaseFromReference returns a BaseComputeResource for a given managed object
// reference.
func BaseFromReference(client *govmomi.Client, ref types.ManagedObjectReference) (BaseComputeResource, error) {
	switch ref.Type {
	case "ComputeResource":
		return StandaloneFromID(client, ref.Value)
	case "ClusterComputeResource":
		return StandaloneFromID(client, ref.Value)
	}
	return nil, fmt.Errorf("unknown object type %s", ref.Type)
}

// BaseProperties returns the base-level ComputeResource managed object for a
// BaseComputeResource, an interface that any base-level ComputeResource and
// derivative object implements.
//
// Note that this does not return any cluster-level attributes.
func BaseProperties(obj BaseComputeResource) (*mo.ComputeResource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	var props mo.ComputeResource
	if err := obj.Properties(ctx, obj.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

// BasePropertiesFromReference combines BaseFromReference and BaseProperties to
// get a base-level ComputeResource managed object for a specific managed
// object reference.
func BasePropertiesFromReference(client *govmomi.Client, ref types.ManagedObjectReference) (*mo.ComputeResource, error) {
	obj, err := BaseFromReference(client, ref)
	if err != nil {
		return nil, err
	}
	return BaseProperties(obj)
}

// DefaultDevicesFromReference fetches the default virtual device list for a
// specific compute resource from a supplied managed object reference.
func DefaultDevicesFromReference(client *govmomi.Client, ref types.ManagedObjectReference, guest string) (object.VirtualDeviceList, error) {
	log.Printf("[DEBUG] Fetching default device list for object reference %q for OS type %q", ref.Value, guest)
	props, err := BasePropertiesFromReference(client, ref)
	if err != nil {
		return nil, err
	}
	b := envbrowse.NewEnvironmentBrowser(client.Client, *props.EnvironmentBrowser)
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	return b.DefaultDevices(ctx, "", nil)
}

// OSFamily uses the compute resource's environment browser to get the OS family
// for a specific guest ID.
func OSFamily(client *govmomi.Client, ref types.ManagedObjectReference, guest string) (string, error) {
	props, err := BasePropertiesFromReference(client, ref)
	if err != nil {
		return "", err
	}
	b := envbrowse.NewEnvironmentBrowser(client.Client, *props.EnvironmentBrowser)
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	return b.OSFamily(ctx, guest)
}
