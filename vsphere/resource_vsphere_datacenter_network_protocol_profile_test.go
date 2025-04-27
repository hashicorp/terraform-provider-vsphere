// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/types"
)

const testAccVSphereDatacenterNetworkProtocolProfileResourceName = "vsphere_datacenter_network_protocol_profile.testProfile"

func testAccCheckVSphereDatacenterNetworkProtocolProfileConfig(name string) string {
	return fmt.Sprintf(`
resource "vsphere_datacenter" "testDC" {
  name = "test-dc"
}

data "vsphere_network" "network" {
  name          = "VM Network"
  datacenter_id = vsphere_datacenter.testDC.id
}

resource "vsphere_datacenter_network_protocol_profile" "testProfile" {
  datacenter_id      = vsphere_datacenter.testDC.id
  network_id         = data.vsphere_network.network.id
  name               = "%s"
  dns_domain         = "example.com"
  dns_search_path    = "example.local"
  host_prefix        = "prefix"
  http_proxy         = "http://proxy.example.com"

  ipv4 {
    subnet                = "192.168.10.0"
    netmask               = "255.255.255.0"
    gateway               = "192.168.10.1"
    dns_servers           = ["8.8.8.8", "8.8.4.4"]
    dhcp_server_available = true
    ip_pool_range         = "192.168.10.100#16"
  }

  ipv6 {
    subnet                = "2001:db8::"
    netmask               = "ffff:ffff:ffff::"
    gateway               = "2001:db8::1"
    dns_servers           = ["2001:4860:4860::8888"]
    dhcp_server_available = true
    ip_pool_range         = "2001:db8::100#30"
  }
}
`, name)
}

// Basic create and destroy test
func TestAccResourceVSphereDatacenterNetworkProtocolProfile_basic(t *testing.T) {
	name := "testProfile"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereDatacenterNetworkProtocolProfileDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckVSphereDatacenterNetworkProtocolProfileConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVSphereDatacenterNetworkProtocolProfileExists(testAccVSphereDatacenterNetworkProtocolProfileResourceName),
				),
			},
		},
	})
}

// Update test: change the name and re-apply
func TestAccResourceVSphereDatacenterNetworkProtocolProfile_update(t *testing.T) {
	name := "testProfile"
	updatedName := "updatedProfile"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereDatacenterNetworkProtocolProfileDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckVSphereDatacenterNetworkProtocolProfileConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVSphereDatacenterNetworkProtocolProfileExists(testAccVSphereDatacenterNetworkProtocolProfileResourceName),
				),
			},
			{
				Config: testAccCheckVSphereDatacenterNetworkProtocolProfileConfig(updatedName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVSphereDatacenterNetworkProtocolProfileExists(testAccVSphereDatacenterNetworkProtocolProfileResourceName),
				),
			},
		},
	})
}

// Verify the IP pool exists in vSphere
func testAccCheckVSphereDatacenterNetworkProtocolProfileExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		client := testAccProvider.Meta().(*Client).vimClient
		dcID := rs.Primary.Attributes["datacenter_id"]
		dc, err := datacenterFromID(client, dcID)
		if err != nil {
			return fmt.Errorf("cannot locate datacenter %q: %s", dcID, err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
		defer cancel()
		resp, err := methods.QueryIpPools(ctx, client.RoundTripper, &types.QueryIpPools{
			This: *client.ServiceContent.IpPoolManager,
			Dc:   dc.Reference(),
		})
		if err != nil {
			return fmt.Errorf("error querying IP pools: %s", err)
		}

		idInt, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("invalid IP pool ID %q: %s", rs.Primary.ID, err)
		}
		for _, p := range resp.Returnval {
			if p.Id == int32(idInt) {
				return nil
			}
		}
		return fmt.Errorf("IP pool %q not found in datacenter %s", rs.Primary.ID, dcID)
	}
}

// Confirm the IP pool is destroyed
func testAccCheckVSphereDatacenterNetworkProtocolProfileDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client).vimClient
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vsphere_datacenter_network_protocol_profile" {
			continue
		}
		dcID := rs.Primary.Attributes["datacenter_id"]
		dc, err := datacenterFromID(client, dcID)
		if err != nil {
			return fmt.Errorf("cannot locate datacenter %q: %s", dcID, err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
		defer cancel()
		resp, err := methods.QueryIpPools(ctx, client.RoundTripper, &types.QueryIpPools{
			This: *client.ServiceContent.IpPoolManager,
			Dc:   dc.Reference(),
		})
		if err != nil {
			return fmt.Errorf("error querying IP pools: %s", err)
		}

		idInt, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("invalid IP pool ID %q: %s", rs.Primary.ID, err)
		}
		for _, p := range resp.Returnval {
			if p.Id == int32(idInt) {
				return fmt.Errorf("IP pool %q still exists", rs.Primary.ID)
			}
		}
	}
	return nil
}
