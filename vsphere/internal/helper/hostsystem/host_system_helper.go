package hostsystem

import (
	"context"
	"fmt"
	"log"

	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
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
