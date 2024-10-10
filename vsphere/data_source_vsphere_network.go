// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/network"
	"github.com/vmware/govmomi/object"
)

func dataSourceVSphereNetwork() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVSphereNetworkRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name or path of the network.",
				Required:    true,
			},
			"datacenter_id": {
				Type:        schema.TypeString,
				Description: "The managed object ID of the datacenter the network is in. This is required if the supplied path is not an absolute path containing a datacenter and there are multiple datacenters in your infrastructure.",
				Optional:    true,
			},
			"type": {
				Type:        schema.TypeString,
				Description: "The managed object type of the network.",
				Computed:    true,
			},
			"distributed_virtual_switch_uuid": {
				Type:        schema.TypeString,
				Description: "Id of the distributed virtual switch of which the port group is a part of",
				Optional:    true,
			},
			"filter": {
				Type:        schema.TypeSet,
				Description: "Apply a filter for the discovered network.",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network_type": {
							Type:         schema.TypeString,
							Description:  "The type of the network (e.g., Network, DistributedVirtualPortgroup, OpaqueNetwork)",
							Optional:     true,
							ValidateFunc: validation.StringInSlice(network.NetworkType, false),
						},
					},
				},
			},
		},
	}
}

func dataSourceVSphereNetworkRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).vimClient

	name := d.Get("name").(string)
	dvSwitchUUID := d.Get("distributed_virtual_switch_uuid").(string)
	var dc *object.Datacenter
	if dcID, ok := d.GetOk("datacenter_id"); ok {
		var err error
		dc, err = datacenterFromID(client, dcID.(string))
		if err != nil {
			return fmt.Errorf("cannot locate datacenter: %s", err)
		}
	}
	var net object.NetworkReference
	var err error

	vimClient := client.Client

	// Read filter from the schema.
	filters := make(map[string]string)
	if v, ok := d.GetOk("filter"); ok {
		filterList := v.(*schema.Set).List()
		if len(filterList) > 0 {
			for key, value := range filterList[0].(map[string]interface{}) {
				filters[key] = value.(string)
			}
		}
	}

	if dvSwitchUUID != "" {
		// Handle distributed virtual switch port group
		net, err = network.FromNameAndDVSUuid(client, name, dc, dvSwitchUUID)
		if err != nil {
			return fmt.Errorf("error fetching DVS network: %s", err)
		}
	} else {
		// Handle standard switch port group
		net, err = network.FromName(vimClient, name, dc, filters) // Pass the *vim25.Client
		if err != nil {
			return fmt.Errorf("error fetching network: %s", err)
		}
	}

	d.SetId(net.Reference().Value)
	_ = d.Set("type", net.Reference().Type)
	return nil
}
