package resourcepool

import (
	"context"
	"fmt"

	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

// FromPathOrDefault returns a ResourcePool via its supplied path.
func FromPathOrDefault(client *govmomi.Client, name string, dc *object.Datacenter) (*object.ResourcePool, error) {
	finder := find.NewFinder(client.Client, false)
	if dc != nil {
		finder.SetDatacenter(dc)
	}

	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	t := client.ServiceContent.About.ApiType
	switch t {
	case "HostAgent":
		return finder.DefaultResourcePool(ctx)
	case "VirtualCenter":
		if name != "" {
			return finder.ResourcePool(ctx, name)
		}
		return finder.DefaultResourcePool(ctx)
	}
	return nil, fmt.Errorf("unsupported ApiType: %s", t)
}

// FromID locates a ResourcePool by its managed object reference ID.
func FromID(client *govmomi.Client, id string) (*object.ResourcePool, error) {
	finder := find.NewFinder(client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  "ResourcePool",
		Value: id,
	}

	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	ds, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("could not find host system with id: %s: %s", id, err)
	}
	return ds.(*object.ResourcePool), nil
}
