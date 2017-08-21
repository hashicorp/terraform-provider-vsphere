package vsphere

import (
	"fmt"
	"time"

	"context"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// hostNetworkSystemFromName locates a HostNetworkSystem from a specified host
// name. The default host system is used if the client is connected to an ESXi
// host, versus vCenter.
func hostNetworkSystemFromName(client *govmomi.Client, host, datacenter string) (*object.HostNetworkSystem, error) {
	finder := find.NewFinder(client.Client, false)

	var hs *object.HostSystem
	var err error
	switch t := client.ServiceContent.About.ApiType; t {
	case "HostAgent":
		dc, err := getDatacenter(client, "")
		if err != nil {
			return nil, fmt.Errorf("could not get datacenter: %s", err)
		}
		finder.SetDatacenter(dc)
		hs, err = finder.DefaultHostSystem(context.TODO())
	case "VirtualCenter":
		dc, err := getDatacenter(client, datacenter)
		if err != nil {
			return nil, fmt.Errorf("could not get datacenter: %s", err)
		}
		finder.SetDatacenter(dc)
		hs, err = finder.HostSystem(context.TODO(), host)
	default:
		return nil, fmt.Errorf("unsupported ApiType: %s", t)
	}
	if err != nil {
		return nil, fmt.Errorf("error loading host system: %s", err)
	}
	return hs.ConfigManager().NetworkSystem(context.TODO())
}

// hostVSwitchFromName locates a host virtual switch from its assigned name and
// host, using the client's default property collector.
func hostVSwitchFromName(client *govmomi.Client, name, host, datacenter string, timeout time.Duration) (*types.HostVirtualSwitch, error) {
	ns, err := hostNetworkSystemFromName(client, host, datacenter)
	if err != nil {
		return nil, fmt.Errorf("error loading network system: %s", err)
	}

	var mns mo.HostNetworkSystem
	pc := client.PropertyCollector()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := pc.RetrieveOne(ctx, ns.Reference(), []string{"networkInfo.vswitch"}, &mns); err != nil {
		return nil, fmt.Errorf("error fetching host network properties: %s", err)
	}

	for _, sw := range mns.NetworkInfo.Vswitch {
		if sw.Name == name {
			return &sw, nil
		}
	}

	return nil, fmt.Errorf("vSwitch %s not found on host %s", name, host)
}

// hostPortGroupFromName locates a host port group from its assigned name and
// host, using the client's default property collector.
func hostPortGroupFromName(client *govmomi.Client, name, host, datacenter string, timeout time.Duration) (*types.HostPortGroup, error) {
	ns, err := hostNetworkSystemFromName(client, host, datacenter)
	if err != nil {
		return nil, fmt.Errorf("error loading network system: %s", err)
	}

	var mns mo.HostNetworkSystem
	pc := client.PropertyCollector()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := pc.RetrieveOne(ctx, ns.Reference(), []string{"networkInfo.portgroup"}, &mns); err != nil {
		return nil, fmt.Errorf("error fetching host network properties: %s", err)
	}

	for _, pg := range mns.NetworkInfo.Portgroup {
		if pg.Spec.Name == name {
			return &pg, nil
		}
	}

	return nil, fmt.Errorf("port group %s not found on host %s", name, host)
}
