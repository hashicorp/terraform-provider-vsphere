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
				Description: "The name of a storage policy associated with the datastore where the container images will be stored.",
			},
			"management_network": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "The name of the management network which the control plane VMs will be connected to.",
				MaxItems:    1,
				Elem:        mgmtNetworkSchema(),
			},
			"content_library": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the subscribed content library.",
			},
			"main_dns": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "List of DNS servers to use on the Kubernetes API server.",
			},
			"worker_dns": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "List of DNS servers to use on the worker nodes.",
			},
			"edge_cluster": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the NSX Edge Cluster.",
			},
			"dvs_uuid": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The UUID (not ID) of the distributed switch.",
			},
			"sizing_hint": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Size of the Kubernetes API server.",
			},
			"egress_cidr": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "CIDR blocks from which NSX assigns IP addresses used for performing SNAT from container IPs to external IPs.",
				Elem:        cidrSchema(),
			},
			"ingress_cidr": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "CIDR blocks from which NSX assigns IP addresses for Kubernetes Ingresses and Kubernetes Services of type LoadBalancer.",
				Elem:        cidrSchema(),
			},
			"pod_cidr": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "CIDR blocks from which Kubernetes allocates pod IP addresses.",
				Elem:        cidrSchema(),
			},
			"service_cidr": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "CIDR block from which Kubernetes allocates service cluster IP addresses.",
				MaxItems:    1,
				MinItems:    1,
				Elem:        cidrSchema(),
			},
			"search_domains": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of DNS search domains.",
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
				Description: "ID of the network. (e.g. a distributed portgroup).",
			},
			"starting_address": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Starting address of the management network range.",
			},
			"subnet_mask": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Subnet mask.",
			},
			"gateway": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Gateway IP address.",
			},
			"address_count": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Number of addresses to allocate. Starts from 'starting_address'",
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
				Description: "Network address.",
			},
			"prefix": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Subnet prefix.",
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
	mainDns := d.Get("main_dns").(string)
	workerDns := d.Get("worker_dns").(string)
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
		MasterDNS:                              []string{mainDns},
		WorkerDNS:                              []string{workerDns},
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
