// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/vmware/terraform-provider-vsphere/vsphere/internal/helper/network"
	"github.com/vmware/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/vmware/terraform-provider-vsphere/vsphere/internal/helper/viapi"
)

const resourceVSphereDatacenterNetworkProtocolProfileName = "vsphere_datacenter_network_protocol_profile"

func resourceVSphereDatacenterNetworkProtocolProfile() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereDatacenterNetworkProtocolProfileCreate,
		Read:   resourceVSphereDatacenterNetworkProtocolProfileRead,
		Update: resourceVSphereDatacenterNetworkProtocolProfileUpdate,
		Delete: resourceVSphereDatacenterNetworkProtocolProfileDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVSphereDatacenterNetworkProtocolProfileImport,
		},

		Schema: map[string]*schema.Schema{
			"datacenter_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"network_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"dns_domain":      {Type: schema.TypeString, Optional: true},
			"host_prefix":     {Type: schema.TypeString, Optional: true},
			"dns_search_path": {Type: schema.TypeString, Optional: true},
			"http_proxy":      {Type: schema.TypeString, Optional: true},
			"ipv4": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"subnet":                {Type: schema.TypeString, Optional: true},
					"netmask":               {Type: schema.TypeString, Optional: true, Default: "255.255.255.0"},
					"gateway":               {Type: schema.TypeString, Optional: true},
					"dns_servers":           {Type: schema.TypeList, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}, Computed: true},
					"dhcp_server_available": {Type: schema.TypeBool, Optional: true, Default: false},
					"ip_pool_range":         {Type: schema.TypeString, Optional: true},
				}},
			},
			"ipv6": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"subnet":                {Type: schema.TypeString, Optional: true},
					"netmmask":              {Type: schema.TypeString, Optional: true, Default: "ffff:ffff:ffff:ffff:ffff:ffff:0:0"},
					"gateway":               {Type: schema.TypeString, Optional: true},
					"dns_servers":           {Type: schema.TypeList, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}, Computed: true},
					"dhcp_server_available": {Type: schema.TypeBool, Optional: true, Default: false},
					"ip_pool_range":         {Type: schema.TypeString, Optional: true},
				}},
			},
		},
	}
}

// Create a new IP pool (network protocol profile)
func resourceVSphereDatacenterNetworkProtocolProfileCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).vimClient
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return err
	}

	// Datacenter lookup
	dcID := d.Get("datacenter_id").(string)
	dc, err := datacenterFromID(client, dcID)
	if err != nil {
		return fmt.Errorf("cannot locate datacenter %q: %s", dcID, err)
	}

	// Network lookup
	netID := d.Get("network_id").(string)
	netObj, err := network.FromID(client, netID)
	if err != nil {
		return fmt.Errorf("cannot locate network %q: %s", netID, err)
	}
	netRef := netObj.Reference()

	// Gather fields
	name := d.Get("name").(string)
	ipv4Ranges := d.Get("ipv4.0.ip_pool_range").(string)
	ipv6Ranges := d.Get("ipv6.0.ip_pool_range").(string)
	ipv4Enabled := len(ipv4Ranges) > 0
	ipv6Enabled := len(ipv6Ranges) > 0

	// Build the pool spec
	pool := types.IpPool{
		Name:          name,
		DnsDomain:     d.Get("dns_domain").(string),
		DnsSearchPath: d.Get("dns_search_path").(string),
		HostPrefix:    d.Get("host_prefix").(string),
		HttpProxy:     d.Get("http_proxy").(string),
		NetworkAssociation: []types.IpPoolAssociation{{
			Network: &netRef,
		}},
		Ipv4Config: &types.IpPoolIpPoolConfigInfo{
			SubnetAddress:       d.Get("ipv4.0.subnet").(string),
			Netmask:             d.Get("ipv4.0.netmask").(string),
			Gateway:             d.Get("ipv4.0.gateway").(string),
			Dns:                 expandStringList(d.Get("ipv4.0.dns_servers").([]interface{})),
			DhcpServerAvailable: structure.GetBoolPtr(d, "ipv4.0.dhcp_server_available"),
			IpPoolEnabled:       &ipv4Enabled,
			Range:               ipv4Ranges,
		},
		Ipv6Config: &types.IpPoolIpPoolConfigInfo{
			SubnetAddress:       d.Get("ipv6.0.subnet").(string),
			Gateway:             d.Get("ipv6.0.gateway").(string),
			Dns:                 expandStringList(d.Get("ipv6.0.dns_servers").([]interface{})),
			DhcpServerAvailable: structure.GetBoolPtr(d, "ipv6.0.dhcp_server_available"),
			IpPoolEnabled:       &ipv6Enabled,
			Range:               ipv6Ranges,
		},
	}

	// Call CreateIpPool
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	if _, err := methods.CreateIpPool(ctx, client.RoundTripper, &types.CreateIpPool{
		This: *client.ServiceContent.IpPoolManager,
		Dc:   dc.Reference(),
		Pool: pool,
	}); err != nil {
		return fmt.Errorf("error creating network protocol profile: %s", err)
	}

	// Query to obtain the new pool's ID
	resp, err := methods.QueryIpPools(ctx, client.RoundTripper, &types.QueryIpPools{
		This: *client.ServiceContent.IpPoolManager,
		Dc:   dc.Reference(),
	})
	if err != nil {
		return fmt.Errorf("error fetching network protocol profile: %s", err)
	}
	var created *types.IpPool
	for _, p := range resp.Returnval {
		if p.Name == name {
			created = &p
			break
		}
	}
	if created == nil {
		return fmt.Errorf("created network protocol profile %q not found", name)
	}
	// Set Terraform ID to the numeric pool ID
	d.SetId(strconv.Itoa(int(created.Id)))

	// Read in the created state
	return resourceVSphereDatacenterNetworkProtocolProfileRead(d, meta)
}

// Read loads the IP pool from vSphere and flattens into state
func resourceVSphereDatacenterNetworkProtocolProfileRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).vimClient
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return err
	}

	// Datacenter lookup
	dcID := d.Get("datacenter_id").(string)
	dc, err := datacenterFromID(client, dcID)
	if err != nil {
		return fmt.Errorf("cannot locate datacenter %q: %s", dcID, err)
	}

	// Query all pools
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	resp, err := methods.QueryIpPools(ctx, client.RoundTripper, &types.QueryIpPools{
		This: *client.ServiceContent.IpPoolManager,
		Dc:   dc.Reference(),
	})
	if err != nil {
		return fmt.Errorf("error querying network protocol profiles: %s", err)
	}

	// Find our pool by ID
	idInt, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("invalid IP pool ID %q: %s", d.Id(), err)
	}
	var pool *types.IpPool
	for _, p := range resp.Returnval {
		if p.Id == int32(idInt) {
			pool = &p
			break
		}
	}
	if pool == nil {
		// Gone, remove from state
		d.SetId("")
		return nil
	}

	// Flatten into Terraform
	d.SetId(strconv.Itoa(int(pool.Id)))
	return flattenIpPool(d, *pool)
}

// Update modifies fields on the existing IP pool
func resourceVSphereDatacenterNetworkProtocolProfileUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).vimClient
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return err
	}

	// Datacenter lookup
	dcID := d.Get("datacenter_id").(string)
	dc, err := datacenterFromID(client, dcID)
	if err != nil {
		return fmt.Errorf("cannot locate datacenter %q: %s", dcID, err)
	}
	// Network lookup
	netID := d.Get("network_id").(string)
	netObj, err := network.FromID(client, netID)
	if err != nil {
		return fmt.Errorf("cannot locate network %q: %s", netID, err)
	}
	netRef := netObj.Reference()

	// Build updated pool
	idInt, _ := strconv.Atoi(d.Id())
	ipv4Ranges := d.Get("ipv4.0.ip_pool_range").(string)
	ipv6Ranges := d.Get("ipv6.0.ip_pool_range").(string)
	ipv4Enabled := len(ipv4Ranges) > 0
	ipv6Enabled := len(ipv6Ranges) > 0

	pool := types.IpPool{
		Id:            int32(idInt),
		Name:          d.Get("name").(string),
		DnsDomain:     d.Get("dns_domain").(string),
		DnsSearchPath: d.Get("dns_search_path").(string),
		HostPrefix:    d.Get("host_prefix").(string),
		HttpProxy:     d.Get("http_proxy").(string),
		NetworkAssociation: []types.IpPoolAssociation{
			{Network: &netRef},
		},
		Ipv4Config: &types.IpPoolIpPoolConfigInfo{
			SubnetAddress:       d.Get("ipv4.0.subnet").(string),
			Netmask:             d.Get("ipv4.0.netmask").(string),
			Gateway:             d.Get("ipv4.0.gateway").(string),
			Dns:                 expandStringList(d.Get("ipv4.0.dns_servers").([]interface{})),
			DhcpServerAvailable: structure.GetBoolPtr(d, "ipv4.0.dhcp_server_available"),
			IpPoolEnabled:       &ipv4Enabled,
			Range:               ipv4Ranges,
		},
		Ipv6Config: &types.IpPoolIpPoolConfigInfo{
			SubnetAddress:       d.Get("ipv6.0.subnet").(string),
			Gateway:             d.Get("ipv6.0.gateway").(string),
			Dns:                 expandStringList(d.Get("ipv6.0.dns_servers").([]interface{})),
			DhcpServerAvailable: structure.GetBoolPtr(d, "ipv6.0.dhcp_server_available"),
			IpPoolEnabled:       &ipv6Enabled,
			Range:               ipv6Ranges,
		},
	}

	// Call UpdateIpPool
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	if _, err := methods.UpdateIpPool(ctx, client.RoundTripper, &types.UpdateIpPool{
		This: *client.ServiceContent.IpPoolManager,
		Dc:   dc.Reference(),
		Pool: pool,
	}); err != nil {
		return fmt.Errorf("error updating network protocol profile: %s", err)
	}

	// Refresh
	return resourceVSphereDatacenterNetworkProtocolProfileRead(d, meta)
}

// Delete destroys the IP pool
func resourceVSphereDatacenterNetworkProtocolProfileDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).vimClient
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return err
	}

	// Datacenter lookup
	dcID := d.Get("datacenter_id").(string)
	dc, err := datacenterFromID(client, dcID)
	if err != nil {
		return fmt.Errorf("cannot locate datacenter %q: %s", dcID, err)
	}

	idInt, _ := strconv.Atoi(d.Id())
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	if _, err := methods.DestroyIpPool(ctx, client.RoundTripper, &types.DestroyIpPool{
		This:  *client.ServiceContent.IpPoolManager,
		Dc:    dc.Reference(),
		Id:    int32(idInt),
		Force: true,
	}); err != nil {
		return fmt.Errorf("error deleting network protocol profile: %s", err)
	}
	d.SetId("")
	return nil
}

// Import state using "<datacenter_id>:<pool_id>"
func resourceVSphereDatacenterNetworkProtocolProfileImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("import ID must be <datacenter_id>:<pool_id>")
	}
	d.Set("datacenter_id", parts[0])
	d.SetId(parts[1])
	return []*schema.ResourceData{d}, nil
}

// flattenIpPool writes a types.IpPool into Terraform state
func flattenIpPool(d *schema.ResourceData, pool types.IpPool) error {
	d.Set("name", pool.Name)
	d.Set("dns_domain", pool.DnsDomain)
	d.Set("dns_search_path", pool.DnsSearchPath)
	d.Set("host_prefix", pool.HostPrefix)
	d.Set("http_proxy", pool.HttpProxy)
	if len(pool.NetworkAssociation) > 0 && pool.NetworkAssociation[0].Network != nil {
		d.Set("network_id", pool.NetworkAssociation[0].Network.Value)
	}
	if pool.Ipv4Config != nil {
		cfg := pool.Ipv4Config
		d.Set("ipv4", []map[string]interface{}{{
			"subnet":                cfg.SubnetAddress,
			"netmask":               cfg.Netmask,
			"gateway":               cfg.Gateway,
			"dns_servers":           cfg.Dns,
			"dhcp_server_available": cfg.DhcpServerAvailable,
			"ip_pool_range":         cfg.Range,
		}})
	}
	if pool.Ipv6Config != nil {
		cfg := pool.Ipv6Config
		d.Set("ipv6", []map[string]interface{}{{
			"subnet":                cfg.SubnetAddress,
			"gateway":               cfg.Gateway,
			"dns_servers":           cfg.Dns,
			"dhcp_server_available": cfg.DhcpServerAvailable,
			"ip_pool_range":         cfg.Range,
		}})
	}
	return nil
}

// expandStringList casts []interface{} to []string
func expandStringList(list []interface{}) []string {
	out := make([]string, 0, len(list))
	for _, v := range list {
		out = append(out, v.(string))
	}
	return out
}
