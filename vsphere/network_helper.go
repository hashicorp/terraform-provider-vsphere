package vsphere

import (
	"context"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
)

// networkFromPath loads a network via its path.
//
// A network is a usually one of three kinds of networks: a DVS port group, a
// host port group, or a "opaque" network, provided externally from something
// like NSX. All three of these can be used as a backing for a virtual ethernet
// card, which is usually what these helpers are used with.
//
// Datacenter is optional here - if not provided, it's expected that the path
// is sufficient enough for finder to determine the datacenter required.
func networkFromPath(client *govmomi.Client, name string, dc *object.Datacenter) (object.NetworkReference, error) {
	finder := find.NewFinder(client.Client, false)
	if dc != nil {
		finder.SetDatacenter(dc)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	return finder.Network(ctx, name)
}

// networkReferenceProperties is a convenience method that wraps fetching the
// Network MO from a NetworkReference.
//
// Note that regardless of the network type, this only fetches the Network MO
// and not any of the extended properties of that network.
func genericNetworkProperties(client *govmomi.Client, net object.NetworkReference) (*mo.Network, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	var props mo.Network
	nc := object.NewCommon(client.Client, net.Reference())
	if err := nc.Properties(ctx, nc.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

// networkProperties gets the properties for a specific Network.
//
// By itself, the Network type usually represents a standard port group in
// vCenter - it has been set up on a host or a set of hosts, and is usually
// configured via through an appropriate HostNetworkSystem. vCenter, however,
// groups up these networks and displays them as a single network that VM can
// use across hosts, facilitating HA and vMotion for VMs that use standard port
// groups versus DVS port groups. Hence the "Network" object is mainly a
// read-only MO and is only useful for checking some very base level
// attributes.
//
// While other network MOs extend the base network object (such as DV port
// groups and opaque networks), this only works with the base object only.
// Refer to functions more specific to the MO to get a fully extended property
// set for the extended objects if you are dealing with those object types.
func networkProperties(net *object.Network) (*mo.Network, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	var props mo.Network
	if err := net.Properties(ctx, net.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}
