package vsphere

import (
	"fmt"
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	// "github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/datastore"
	// "github.com/vmware/govmomi/object"
	// "github.com/vmware/govmomi/property"
	// gtask "github.com/vmware/govmomi/task"
	// "github.com/vmware/govmomi/vim25"
	// "github.com/vmware/govmomi"
	// "github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/methods"
	// "github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func dataSourceVSphereVcha() *schema.Resource {
	return &schema.Resource {
		Read: dataSourceVSphereVchaRead,

		Schema: map[string]*schema.Schema{
			"datacenter": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the vSphere datacenter where vCHA is to be deployed.",
			},
			"network": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "VCHA network name",
			},
			"netmask": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "VCHA network mask",
			},
			"address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Active node address",
			},
			"username": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Active node username",
			},
			"password": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Active node password",
			},
			"thumbprint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Active node SSL thumbprint",
			},
			"active_ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Active node IP address",
			},
			"active_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Active node VM name",
			},
			"passive_datastore": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Passive node datastore name",
			},
			"passive_ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Passive node IP address",
			},
			"passive_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Passive node VM name",
			},
			"witness_datastore": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Witness node datastore name",
			},
			"witness_ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Witness node IP address",
			},
			"witness_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Witness node VM name",
			},
			/*
				"active": {
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{Schema: map[string]*schema.Schema{
						"thumbprint": {
							Type:        schema.TypeBool,
							Optional:    false,
							Description: "Active node thumbprint",
						},
						"ip": {
							Type:        schema.TypeString,
							Optional:    false,
							Description: "Active node IP address",
						},
						"name": {
							Type:        schema.TypeString,
							Optional:    false,
							Description: "Active node VM name",
						},
					}},
				},
				"passive": {
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{Schema: map[string]*schema.Schema{
						"datastore": {
							Type:        schema.TypeBool,
							Optional:    false,
							Description: "Active node thumbprint",
						},
						"ip": {
							Type:        schema.TypeString,
							Optional:    false,
							Description: "Active node IP address",
						},
						"name": {
							Type:        schema.TypeString,
							Optional:    false,
							Description: "Active node VM name",
						},
					}},
				},
				"witness": {
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{Schema: map[string]*schema.Schema{
						"datastore": {
							Type:        schema.TypeBool,
							Optional:    false,
							Description: "Active node thumbprint",
						},
						"ip": {
							Type:        schema.TypeString,
							Optional:    false,
							Description: "Active node IP address",
						},
						"name": {
							Type:        schema.TypeString,
							Optional:    false,
							Description: "Active node VM name",
						},
					}},
				},
			*/
		},
	}
}

func dataSourceVSphereVchaRead(d *schema.ResourceData, meta interface{}) error {
	// NOTE: Destroying the host without telling vsphere about it will result in us not
	// knowing that the host does not exist any more.

	// Look for host
	client := meta.(*Client).vimClient
	// hostID := d.Id()

	getVchaConfig := types.GetVchaConfig{
		This: *client.Client.ServiceContent.FailoverClusterConfigurator,
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()

	taskResponse, err := methods.GetVchaConfig(ctx, client.Client, &getVchaConfig)
	if err != nil {
		return fmt.Errorf("error while retrieving vCHA.  Error: %s", err)
	}

	vchaClusterConfigInfo := taskResponse.Returnval
	failoverNodeInfo1 := vchaClusterConfigInfo.FailoverNodeInfo1
	failoverNode1Ip := failoverNodeInfo1.ClusterIpSettings.Ip.(*types.CustomizationFixedIp)
	err = d.Set("active_ip", failoverNode1Ip.IpAddress)
	if err != nil {
		return err
	}
	failoverNodeInfo2 := vchaClusterConfigInfo.FailoverNodeInfo2
	failoverNode2Ip := failoverNodeInfo2.ClusterIpSettings.Ip.(*types.CustomizationFixedIp)
	err = d.Set("passive_ip", failoverNode2Ip.IpAddress)
	if err != nil {
		return err
	}
	witnessNodeInfo := vchaClusterConfigInfo.WitnessNodeInfo
	witnessIp := witnessNodeInfo.IpSettings.Ip.(*types.CustomizationFixedIp)
	err = d.Set("witness_ip", witnessIp.IpAddress)
	if err != nil {
		return err
	}
	d.SetId(failoverNodeInfo1.BiosUuid)
	return nil
}
