// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/vmware/govmomi/vapi/namespace"
	"time"
)

func resourceVsphereSupervisor() *schema.Resource {
	return &schema.Resource{
		Create: resourceVsphereSupervisorCreate,
		Read:   resourceVsphereSupervisorRead,
		Update: resourceVsphereSupervisorUpdate,
		Delete: resourceVsphereSupervisorDelete,
		Schema: map[string]*schema.Schema{
			"cluster": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the vSphere cluster on which workload management will be enabled.",
			},
			"storage_policy": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "TODO",
			},
			"management_network": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "TODO",
				MaxItems:    1,
				Elem:        mgmtNetworkSchema(),
			},
			"content_library": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "TODO",
			},
			"dns": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "TODO",
			},
			"edge_cluster": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "TODO",
			},
			"dvs_uuid": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "TODO",
			},
			"sizing_hint": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "TODO",
			},
			"egress_cidr": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "TODO",
				Elem:        cidrSchema(),
			},
			"ingress_cidr": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "TODO",
				Elem:        cidrSchema(),
			},
			"pod_cidr": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "TODO",
				Elem:        cidrSchema(),
			},
			"service_cidr": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "TODO",
				MaxItems:    1,
				MinItems:    1,
				Elem:        cidrSchema(),
			},
			"search_domains": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "TODO",
				MaxItems:    1,
				MinItems:    1,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func mgmtNetworkSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"network": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "TODO",
			},
			"starting_address": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "TODO",
			},
			"subnet_mask": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "TODO",
			},
			"gateway": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "TODO",
			},
			"address_count": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "TODO",
			},
		},
	}
}

func cidrSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"address": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "TODO",
			},
			"prefix": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "TODO",
			},
		},
	}
}

func resourceVsphereSupervisorCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client).restClient
	m := namespace.NewManager(c)

	clusterId := d.Get("cluster").(string)

	spec := buildClusterEnableSpec(d)

	if err := m.EnableCluster(context.Background(), clusterId, spec); err != nil {
		return err
	}

	d.SetId(clusterId)

	return waitForSupervisorEnable(m, d)
}

func resourceVsphereSupervisorRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client).restClient
	m := namespace.NewManager(c)

	cluster := getClusterById(m, d.Id())

	if cluster == nil {
		return fmt.Errorf("could not find cluster %s", cluster.ID)
	}

	return nil
}

func resourceVsphereSupervisorUpdate(d *schema.ResourceData, meta interface{}) error {
	return fmt.Errorf("updating a supervisor's settings is not supported")
}

func resourceVsphereSupervisorDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client).restClient
	m := namespace.NewManager(c)

	return m.DisableCluster(context.Background(), d.Id())
}

func buildClusterEnableSpec(d *schema.ResourceData) *namespace.EnableClusterSpec {
	egressCidrs := d.Get("egress_cidr").([]interface{})
	ingressCidrs := d.Get("ingress_cidr").([]interface{})
	podCidrs := d.Get("pod_cidr").([]interface{})
	ncpNetworkSpec := namespace.NcpClusterNetworkSpec{
		NsxEdgeCluster:           d.Get("edge_cluster").(string),
		ClusterDistributedSwitch: d.Get("dvs_uuid").(string),
		EgressCidrs:              getCidrs(egressCidrs),
		IngressCidrs:             getCidrs(ingressCidrs),
		PodCidrs:                 getCidrs(podCidrs),
	}

	contentLib := d.Get("content_library").(string)
	dns := d.Get("dns").(string)
	dnsSearchDomains := d.Get("search_domains").([]interface{})
	storagePolicy := d.Get("storage_policy").(string)
	serviceCidrs := d.Get("service_cidr").([]interface{})

	spec := &namespace.EnableClusterSpec{
		EphemeralStoragePolicy: storagePolicy,
		SizeHint:               getSizingHint(d.Get("sizing_hint")),
		ServiceCidr:            &getCidrs(serviceCidrs)[0],
		// Only NSX-T backing is supported for now
		NetworkProvider:                        &namespace.NsxtContainerPluginNetworkProvider,
		MasterStoragePolicy:                    storagePolicy,
		MasterManagementNetwork:                getMgmtNetwork(d),
		ImageStorage:                           namespace.ImageStorageSpec{StoragePolicy: storagePolicy},
		NcpClusterNetworkSpec:                  &ncpNetworkSpec,
		MasterDNS:                              []string{dns},
		WorkerDNS:                              []string{dns},
		DefaultKubernetesServiceContentLibrary: contentLib,
		MasterDNSSearchDomains:                 structure.SliceInterfacesToStrings(dnsSearchDomains),
	}
	return spec
}

func getMgmtNetwork(d *schema.ResourceData) *namespace.MasterManagementNetwork {
	mgmtNetworkProperty := d.Get("management_network").([]interface{})
	mgmtNetworkData := mgmtNetworkProperty[0].(map[string]interface{})
	return &namespace.MasterManagementNetwork{
		Mode:    &namespace.StaticRangeIpAssignmentMode,
		Network: mgmtNetworkData["network"].(string),
		AddressRange: &namespace.AddressRange{
			SubnetMask:      mgmtNetworkData["subnet_mask"].(string),
			StartingAddress: mgmtNetworkData["starting_address"].(string),
			Gateway:         mgmtNetworkData["gateway"].(string),
			AddressCount:    mgmtNetworkData["address_count"].(int),
		},
	}
}

func getCidrs(data []interface{}) []namespace.Cidr {
	result := make([]namespace.Cidr, len(data))
	for i, cidrData := range data {
		cidr := cidrData.(map[string]interface{})
		result[i] = namespace.Cidr{
			Address: cidr["address"].(string),
			Prefix:  cidr["prefix"].(int),
		}
	}
	return result
}

func getSizingHint(data interface{}) *namespace.SizingHint {
	switch data {
	case "TINY":
		return &namespace.TinySizingHint
	case "SMALL":
		return &namespace.SmallSizingHint
	case "MEDIUM":
		return &namespace.MediumSizingHint
	case "LARGE":
		return &namespace.LargeSizingHint
	}

	return &namespace.UndefinedSizingHint
}

func waitForSupervisorEnable(m *namespace.Manager, d *schema.ResourceData) error {
	ticker := time.NewTicker(time.Minute * time.Duration(1))

	for {
		select {
		case <-context.Background().Done():
		case <-ticker.C:
			cluster := getClusterById(m, d.Id())

			if cluster == nil {
				return fmt.Errorf("could not find cluster %s", cluster.ID)
			}

			if namespace.RunningConfigStatus == *cluster.ConfigStatus {
				return nil
			}
			if namespace.ErrorConfigStatus == *cluster.ConfigStatus {
				return fmt.Errorf("could not enable supervisor on cluster %s", cluster.ID)
			}
		}
	}
}

func getClusterById(m *namespace.Manager, id string) *namespace.ClusterSummary {
	clusters, err := m.ListClusters(context.Background())

	if err != nil {
		return nil
	}

	for _, cluster := range clusters {
		if id == cluster.ID {
			return &cluster
		}
	}

	return nil
}
