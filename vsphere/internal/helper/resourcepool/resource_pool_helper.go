package resourcepool

import (
	"context"
	"fmt"
	"log"

	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/computeresource"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// FromPathOrDefault returns a ResourcePool via its supplied path.
func FromPathOrDefault(client *govmomi.Client, name string, dc *object.Datacenter) (*object.ResourcePool, error) {
	finder := find.NewFinder(client.Client, false)

	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	t := client.ServiceContent.About.ApiType
	switch t {
	case "HostAgent":
		ddc, err := finder.DefaultDatacenter(ctx)
		if err != nil {
			return nil, err
		}
		finder.SetDatacenter(ddc)
		return finder.DefaultResourcePool(ctx)
	case "VirtualCenter":
		if dc != nil {
			finder.SetDatacenter(dc)
		}
		if name != "" {
			return finder.ResourcePool(ctx, name)
		}
		return finder.DefaultResourcePool(ctx)
	}
	return nil, fmt.Errorf("unsupported ApiType: %s", t)
}

// FromID locates a ResourcePool by its managed object reference ID.
func FromID(client *govmomi.Client, id string) (*object.ResourcePool, error) {
	log.Printf("[DEBUG] Locating resource pool with ID %s", id)
	finder := find.NewFinder(client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  "ResourcePool",
		Value: id,
	}

	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	obj, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] Resource pool found: %s", obj.Reference().Value)
	return obj.(*object.ResourcePool), nil
}

// Properties returns the ResourcePool managed object from its higher-level
// object.
func Properties(obj *object.ResourcePool) (*mo.ResourcePool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	var props mo.ResourcePool
	if err := obj.Properties(ctx, obj.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

// ValidateHost checks to see if a HostSystem is a member of a ResourcePool
// through cluster membership, or if the HostSystem ID matches the ID of a
// standalone host ComputeResource. An error is returned if it is not a member
// of the cluster to which the resource pool belongs, or if there was some sort
// of other error with checking.
//
// This is used as an extra validation before a VM creation happens, or vMotion
// to a specific host is attempted.
func ValidateHost(client *govmomi.Client, pool *object.ResourcePool, host *object.HostSystem) error {
	if host == nil {
		// Nothing to validate here, move along
		log.Printf("[DEBUG] ValidateHost: no host supplied, nothing to do")
		return nil
	}
	log.Printf("[DEBUG] Validating that host %q is a member of resource pool %q", host.Reference().Value, pool.Reference().Value)
	pprops, err := Properties(pool)
	if err != nil {
		return err
	}
	cprops, err := computeresource.BasePropertiesFromReference(client, pprops.Owner)
	if err != nil {
		return err
	}
	for _, href := range cprops.Host {
		if href.Value == host.Reference().Value {
			log.Printf("[DEBUG] Validated that host %q is a member of resource pool %q.", host.Reference().Value, pool.Reference().Value)
			return nil
		}
	}
	return fmt.Errorf("host ID %q is not a member of resource pool %q", host.Reference().Value, pool.Reference().Value)
}

// DefaultDevices loads a default VirtualDeviceList for a supplied pool
// and guest ID (guest OS type).
func DefaultDevices(client *govmomi.Client, pool *object.ResourcePool, guest string) (object.VirtualDeviceList, error) {
	log.Printf("[DEBUG] Fetching default device list for resource pool %q for OS type %q", pool.Reference().Value, guest)
	pprops, err := Properties(pool)
	if err != nil {
		return nil, err
	}
	return computeresource.DefaultDevicesFromReference(client, pprops.Owner, guest)
}

// OSFamily uses the resource pool's environment browser to get the OS family
// for a specific guest ID.
func OSFamily(client *govmomi.Client, pool *object.ResourcePool, guest string) (string, error) {
	log.Printf("[DEBUG] Looking for OS family for guest ID %q", guest)
	pprops, err := Properties(pool)
	if err != nil {
		return "", err
	}
	return computeresource.OSFamily(client, pprops.Owner, guest)
}
