// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package datacenter

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/folder"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
)

// FromPath returns a Datacenter via its supplied path.
func FromPath(client *govmomi.Client, path string) (*object.Datacenter, error) {
	finder := find.NewFinder(client.Client, false)

	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	return finder.Datacenter(ctx, path)
}

// FromInventoryPath returns the Datacenter object which is part of a given InventoryPath
func FromInventoryPath(client *govmomi.Client, inventoryPath string) (*object.Datacenter, error) {
	dcPath, err := folder.RootPathParticleDatastore.SplitDatacenter(inventoryPath)
	if err != nil {
		return nil, err
	}
	dc, err := FromPath(client, dcPath)
	if err != nil {
		return nil, err
	}

	return dc, nil
}

// DatacenterFromVMInventoryPath returns the Datacenter object which is part of a given VM InventoryPath.
// This also deals with the case where the datacenter is in a folder.
// VM inventory paths look like this: /SomeFolder/DatacenterName/.../VMName (datacenter is in a folder) or
// /DatacenterName/.../VMName (datacenter is not in a folder).
// The function takes the inventory path and splits it by / as the delimiter. It then takes each part
// of the inventory path in turn and tries to get the datacenter of the given name.
// If it doesn't work, the current part is probably a folder and not a datacenter, so it tries again
// with the next part of the inventory path until it gets the datacenter.
// Note that it only tries to find the datacenter on the given VM inventory path, and NOT on other paths.
// So, datacenters with the same name as another folder should not be a problem.
// If the datacenter is not found at all on the given inventory path, it returns an error.
func DatacenterFromVMInventoryPath(client *govmomi.Client, inventoryPath string) (*object.Datacenter, error) {
	// Split the VM inventory path by /.
	inventoryPathSplit := strings.Split(inventoryPath, "/")

	// The datacenter name is in the inventory path.
	// Iterate through the inventory path until we find the datacenter name.
	for _, dcAttempt := range inventoryPathSplit {
		dc, err := FromPath(client, dcAttempt)
		if err != nil {
			// Datacenter with the given name not found, it is probably a folder, so try again with the next part of the inventory path.
			log.Printf("[DEBUG] Could not find datacenter '%s' on inventory path %s, trying again", dcAttempt, inventoryPath)
		} else {
			// We got the datacenter, return it.
			log.Printf("[DEBUG] Found datacenter %s on inventory path %s", dcAttempt, inventoryPath)
			return dc, nil
		}
	}
	// If we still don't have the datacenter, return an error.
	return nil, fmt.Errorf("Could not find datacenter on inventory path %s", inventoryPath)
}
