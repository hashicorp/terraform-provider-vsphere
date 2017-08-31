package vsphere

import (
	"context"
	"fmt"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// hostNetworkSystemFromHostSystem locates a HostNetworkSystem from a specified
// HostSystem.
func hostNetworkSystemFromHostSystem(hs *object.HostSystem) (*object.HostNetworkSystem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	return hs.ConfigManager().NetworkSystem(ctx)
}

// hostNetworkSystemFromHostSystemID locates a HostNetworkSystem from a
// specified HostSystem managed object ID.
func hostNetworkSystemFromHostSystemID(client *govmomi.Client, hsID string) (*object.HostNetworkSystem, error) {
	hs, err := hostSystemFromID(client, hsID)
	if err != nil {
		return nil, err
	}
	return hostNetworkSystemFromHostSystem(hs)
}

// hostVSwitchFromName locates a virtual switch on the supplied HostSystem by
// name.
func hostVSwitchFromName(client *govmomi.Client, ns *object.HostNetworkSystem, name string) (*types.HostVirtualSwitch, error) {
	var mns mo.HostNetworkSystem
	pc := client.PropertyCollector()
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	if err := pc.RetrieveOne(ctx, ns.Reference(), []string{"networkInfo.vswitch"}, &mns); err != nil {
		return nil, fmt.Errorf("error fetching host network properties: %s", err)
	}

	for _, sw := range mns.NetworkInfo.Vswitch {
		if sw.Name == name {
			return &sw, nil
		}
	}

	return nil, fmt.Errorf("could not find virtual switch %s", name)
}
