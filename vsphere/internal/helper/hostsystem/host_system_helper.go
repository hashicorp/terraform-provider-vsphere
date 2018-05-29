package hostsystem

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/viapi"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// SystemOrDefault returns a HostSystem from a specific host name and
// datacenter. If the user is connecting over ESXi, the default host system is
// used.
func SystemOrDefault(client *govmomi.Client, name string, dc *object.Datacenter) (*object.HostSystem, error) {
	finder := find.NewFinder(client.Client, false)
	finder.SetDatacenter(dc)

	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	t := client.ServiceContent.About.ApiType
	switch t {
	case "HostAgent":
		return finder.DefaultHostSystem(ctx)
	case "VirtualCenter":
		if name != "" {
			return finder.HostSystem(ctx, name)
		}
		return finder.DefaultHostSystem(ctx)
	}
	return nil, fmt.Errorf("unsupported ApiType: %s", t)
}

// FromID locates a HostSystem by its managed object reference ID.
func FromID(client *govmomi.Client, id string) (*object.HostSystem, error) {
	log.Printf("[DEBUG] Locating host system ID %s", id)
	finder := find.NewFinder(client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  "HostSystem",
		Value: id,
	}

	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	hs, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("could not find host system with id: %s: %s", id, err)
	}
	log.Printf("[DEBUG] Host system found: %s", hs.Reference().Value)
	return hs.(*object.HostSystem), nil
}

// Properties is a convenience method that wraps fetching the HostSystem MO
// from its higher-level object.
func Properties(host *object.HostSystem) (*mo.HostSystem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	var props mo.HostSystem
	if err := host.Properties(ctx, host.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

// ResourcePool is a convenience method that wraps fetching the host system's
// root resource pool
func ResourcePool(host *object.HostSystem) (*object.ResourcePool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	return host.ResourcePool(ctx)
}

// hostSystemNameFromID returns the name of a host via its its managed object
// reference ID.
func hostSystemNameFromID(client *govmomi.Client, id string) (string, error) {
	hs, err := FromID(client, id)
	if err != nil {
		return "", err
	}
	return hs.Name(), nil
}

// NameOrID is a convenience method mainly for helping displaying friendly
// errors where space is important - it displays either the host name or the ID
// if there was an error fetching it.
func NameOrID(client *govmomi.Client, id string) string {
	name, err := hostSystemNameFromID(client, id)
	if err != nil {
		return id
	}
	return name
}

// EnterMaintenanceMode puts a host into maintenance mode. If evacuate is set
// to true, all powered off VMs will be removed from the host, or the task will
// block until this is the case, depending on whether or not DRS is on or off
// for the host's cluster. This parameter is ignored on direct ESXi.
func EnterMaintenanceMode(host *object.HostSystem, timeout int, evacuate bool) error {
	if err := viapi.VimValidateVirtualCenter(host.Client()); err != nil {
		evacuate = false
	}

	log.Printf("[DEBUG] Host %q is entering maintenance mode (evacuate: %t)", host.Name(), evacuate)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
	defer cancel()
	task, err := host.EnterMaintenanceMode(ctx, int32(timeout), evacuate, nil)
	if err != nil {
		return err
	}

	return task.Wait(ctx)
}

// ExitMaintenanceMode takes a host out of maintenance mode.
func ExitMaintenanceMode(host *object.HostSystem, timeout int) error {
	log.Printf("[DEBUG] Host %q is exiting maintenance mode", host.Name())

	// Add 5 minutes to timeout for the context timeout to allow for any issues
	// with the request after.
	// TODO: Fix this so that it ultimately uses the provider context.
	ctxTimeout := timeout + 300
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(ctxTimeout))
	defer cancel()
	task, err := host.ExitMaintenanceMode(ctx, int32(timeout))
	if err != nil {
		return err
	}

	return task.Wait(ctx)
}
