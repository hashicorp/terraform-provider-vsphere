// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/datastore"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/storagepod"
	"github.com/vmware/govmomi/object"
)

func dataSourceVSphereDatastoreStats() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVSphereDatastoreStatsRead,

		Schema: map[string]*schema.Schema{
			"datacenter_id": {
				Type:        schema.TypeString,
				Description: "The managed object ID of the datacenter to get datastores from.",
				Required:    true,
			},
			"free_space": {
				Type:        schema.TypeMap,
				Description: "The free space of the datastores.",
				Optional:    true,
			},
			"capacity": {
				Type:        schema.TypeMap,
				Description: "The capacity of the datastores.",
				Optional:    true,
			},
		},
	}
}

func dataSourceVSphereDatastoreStatsRead(d *schema.ResourceData, meta interface{}) error {
	ctx := context.Background()
	client := meta.(*Client).vimClient
	var dc *object.Datacenter
	if dcID, ok := d.GetOk("datacenter_id"); ok {
		var err error
		dc, err = datacenterFromID(client, dcID.(string))
		if err != nil {
			return fmt.Errorf("cannot locate datacenter: %s", err)
		}
	}
	dss, err := datastore.List(client)
	if err != nil {
		return fmt.Errorf("error listing datastores: %s", err)
	}
	storagePods, err := storagepod.List(client)
	if err != nil {
		return fmt.Errorf("error retrieving storage pods: %s", err)
	}
	for s := range storagePods {
		childDatastores, err := storagePods[s].Children(ctx)
		if err != nil {
			return fmt.Errorf("error retrieving datastores in datastore cluster: %s", err)
		}
		for c := range childDatastores {
			ds, err := datastore.FromID(client, childDatastores[c].Reference().Value)
			if err != nil {
				return fmt.Errorf("error retrieving datastore: %s", err)
			}
			dss = append(dss, ds)
		}
	}
	for i := range dss {
		ds, err := datastore.FromPath(client, dss[i].Name(), dc)
		if err != nil {
			return fmt.Errorf("error fetching datastore: %s", err)
		}
		props, err := datastore.Properties(ds)
		if err != nil {
			return fmt.Errorf("error getting properties for datastore ID %q: %s", ds.Reference().Value, err)
		}
		cap := d.Get("capacity").(map[string]interface{})
		cap[dss[i].Name()] = fmt.Sprintf("%v", props.Summary.Capacity)
		d.Set("capacity", cap)
		fr := d.Get("free_space").(map[string]interface{})
		fr[dss[i].Name()] = fmt.Sprintf("%v", props.Summary.FreeSpace)
		d.Set("free_space", fr)
	}
	d.SetId(fmt.Sprintf("%s_stats", dc.Reference().Value))
	return nil
}
