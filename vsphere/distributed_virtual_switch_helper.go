package vsphere

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// Get a list of ManagedObjectReference to HostSystem to use in creating the uplinks for the DVS
func getHostSystemManagedObjectReference(d *schema.ResourceData, client *govmomi.Client) (map[string]types.ManagedObjectReference, error) {
	mor := make(map[string]types.ManagedObjectReference)

	if v, ok := d.GetOk("host"); ok {
		for _, vi := range v.([]interface{}) {
			hi := vi.(map[string]interface{})
			hsID := hi["host_system_id"].(string)

			h, err := hostSystemFromID(client, hsID)
			if err != nil {
				return nil, err
			}
			mor[hsID] = h.Common.Reference()
		}
	}
	return mor, nil
}

// Check if a DVS exists and return a reference to it in case it does
func dvsExists(d *schema.ResourceData, meta interface{}) (object.NetworkReference, error) {
	client := meta.(*govmomi.Client)
	name := d.Get("name").(string)

	dc, err := getDatacenter(client, d.Get("datacenter").(string))
	if err != nil {
		return nil, err
	}

	finder := find.NewFinder(client.Client, true)
	finder = finder.SetDatacenter(dc)

	dvs, err := finder.Network(context.TODO(), name)
	return dvs, err
}

func dvsFromName(client *govmomi.Client, datacenter, name string) (*mo.DistributedVirtualSwitch, error) {
	dc, err := getDatacenter(client, datacenter)
	if err != nil {
		return nil, err
	}

	finder := find.NewFinder(client.Client, true)
	finder = finder.SetDatacenter(dc)

	dvs, err := finder.Network(context.TODO(), name)

	var mdvs mo.DistributedVirtualSwitch
	pc := client.PropertyCollector()
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	if err := pc.RetrieveOne(ctx, dvs.Reference(), nil, &mdvs); err != nil {
		return nil, fmt.Errorf("error fetching uuid property: %s", err)
	}
	return &mdvs, nil
}

func dvsFromUuid(client *govmomi.Client, uuid string) (*mo.DistributedVirtualSwitch, error) {
	dvsm := types.ManagedObjectReference{Type: "DistributedVirtualSwitchManager", Value: "DVSManager"}
	req := &types.QueryDvsByUuid{
		This: dvsm,
		Uuid: uuid,
	}
	dvs, err := methods.QueryDvsByUuid(context.TODO(), client, req)
	if err != nil {
		return nil, fmt.Errorf("error fetching dvs from uuid: %s", err)
	}

	var mdvs mo.DistributedVirtualSwitch
	pc := client.PropertyCollector()
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	if err := pc.RetrieveOne(ctx, dvs.Returnval.Reference(), nil, &mdvs); err != nil {
		return nil, fmt.Errorf("error fetching distributed virtual switch: %s", err)
	}
	return &mdvs, nil
}

// check if host is in refs with the help of hosts
// not very efficient, but the number of entries is usually pretty small
func isHostPartOfDVS(hosts []interface{}, refs map[string]types.ManagedObjectReference, host *types.ManagedObjectReference) map[string]interface{} {
	for _, h := range hosts {
		hi := h.(map[string]interface{})
		if val, ok := refs[hi["host_system_id"].(string)]; ok {
			if val == *host {
				return hi
			}
		}
	}
	return nil
}
