package vsphere

import (
	"context"
	"fmt"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// dvPortgroupFromKey gets a portgroup object from its key.
func dvPortgroupFromKey(client *govmomi.Client, dvsUUID, pgKey string) (*object.DistributedVirtualPortgroup, error) {
	dvsm := types.ManagedObjectReference{Type: "DistributedVirtualSwitchManager", Value: "DVSManager"}
	req := &types.DVSManagerLookupDvPortGroup{
		This:         dvsm,
		SwitchUuid:   dvsUUID,
		PortgroupKey: pgKey,
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	resp, err := methods.DVSManagerLookupDvPortGroup(ctx, client, req)
	if err != nil {
		return nil, err
	}

	return dvPortgroupFromMOID(client, resp.Returnval.Reference().Value)
}

// dvPortgroupFromMOID locates a portgroup by its managed object reference ID.
func dvPortgroupFromMOID(client *govmomi.Client, id string) (*object.DistributedVirtualPortgroup, error) {
	finder := find.NewFinder(client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  "DistributedVirtualPortgroup",
		Value: id,
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	ds, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		return nil, err
	}
	// Should be safe to return here. If our reference returned here and is not a
	// DistributedVirtualPortgroup, then we have bigger problems and to be
	// honest we should be panicking anyway.
	return ds.(*object.DistributedVirtualPortgroup), nil
}

// dvPortgroupFromPath gets a portgroup object from its path.
func dvPortgroupFromPath(client *govmomi.Client, name string, dc *object.Datacenter) (*object.DistributedVirtualPortgroup, error) {
	finder := find.NewFinder(client.Client, false)
	if dc != nil {
		finder.SetDatacenter(dc)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	net, err := finder.Network(ctx, name)
	if err != nil {
		return nil, err
	}
	if net.Reference().Type != "DistributedVirtualPortgroup" {
		return nil, fmt.Errorf("network at path %q is not a portgroup (type %s)", name, net.Reference().Type)
	}
	return dvPortgroupFromMOID(client, net.Reference().Value)
}

// dvPortgroupProperties is a convenience method that wraps fetching the
// portgroup MO from its higher-level object.
func dvPortgroupProperties(pg *object.DistributedVirtualPortgroup) (*mo.DistributedVirtualPortgroup, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	var props mo.DistributedVirtualPortgroup
	if err := pg.Properties(ctx, pg.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

// createDVPortgroup exposes the CreateDVPortgroup_Task method of the
// DistributedVirtualSwitch MO.  This local implementation may go away if this
// is exposed in the higher-level object upstream.
func createDVPortgroup(client *govmomi.Client, dvs *object.VmwareDistributedVirtualSwitch, spec types.DVPortgroupConfigSpec) (*object.Task, error) {
	req := &types.CreateDVPortgroup_Task{
		This: dvs.Reference(),
		Spec: spec,
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	resp, err := methods.CreateDVPortgroup_Task(ctx, client, req)
	if err != nil {
		return nil, err
	}

	return object.NewTask(client.Client, resp.Returnval.Reference()), nil
}
