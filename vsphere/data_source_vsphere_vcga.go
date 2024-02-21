package vsphere

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/types"
)

func dataSourceVSphereVcha() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVSphereVchaRead,

		Schema: map[string]*schema.Schema{
			"datacenter": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the vSphere datacenter where vCHA is to be deployed.",
			},
			"healthstatus": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Health of vCenter HA",
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
			"check_type": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
					Elem: &schema.Schema{Type: schema.TypeString},
				},
				Description: "type of object",
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
		},
	}
}

func dataSourceVSphereVchaRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*Client).vimClient
	getVchaConfig := types.GetVchaConfig{
		This: *client.Client.ServiceContent.FailoverClusterConfigurator,
	}
	getVchaHealth := types.GetVchaClusterHealth{
		This: *client.Client.ServiceContent.FailoverClusterManager,
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

	vchaHealthResponse, err := methods.GetVchaClusterHealth(ctx, client.Client, &getVchaHealth)
	if err != nil {
		return fmt.Errorf("error while retrieving vCHA Health.  Error: %s", err)
	}
	vchaHealthInfo := vchaHealthResponse.Returnval
	clusterInfo := vchaHealthInfo.RuntimeInfo
	health := clusterInfo.ClusterState

	d.Set("healthstatus", health)

	d.SetId("vcha_output")

	return nil
}
