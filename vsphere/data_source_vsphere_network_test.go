package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccDataSourceVSphereNetwork_dvsPortgroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereNetworkPreCheck(t)
			testAccSkipIfEsxi(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereNetworkConfigDVSPortgroup(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.vsphere_network.net", "type", "DistributedVirtualPortgroup"),
					resource.TestCheckResourceAttrPair(
						"data.vsphere_network.net", "id",
						"vsphere_distributed_port_group.pg", "id",
					),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereNetwork_absolutePathNoDatacenter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereNetworkPreCheck(t)
			testAccSkipIfEsxi(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereNetworkConfigDVSPortgroupAbsolute(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.vsphere_network.net", "type", "DistributedVirtualPortgroup"),
					resource.TestCheckResourceAttrPair(
						"data.vsphere_network.net", "id",
						"vsphere_distributed_port_group.pg", "id",
					),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereNetwork_hostPortgroups(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereNetworkPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereNetworkConfigHostPortgroup(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.vsphere_network.net", "type", "Network"),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereNetwork_absolutePathEndingInSameName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereNetworkPreCheck(t)
			testAccSkipIfEsxi(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereNetworkConfigSimilarNet(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.vsphere_network.net", "id",
						"vsphere_distributed_port_group.pg1", "id",
					),
				),
			},
		},
	})
}

func testAccDataSourceVSphereNetworkPreCheck(t *testing.T) {
	if os.Getenv("TF_VAR_VSPHERE_PG_NAME") == "" {
		t.Skip("set TF_VAR_VSPHERE_PG_NAME to run vsphere_network acceptance tests")
	}
}

func testAccDataSourceVSphereNetworkConfigDVSPortgroup() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_distributed_virtual_switch" "dvs" {
  name          = "terraform-test-dvs"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_distributed_port_group" "pg" {
  name                            = "terraform-test-pg"
  distributed_virtual_switch_uuid = "${vsphere_distributed_virtual_switch.dvs.id}"
}

data "vsphere_network" "net" {
  name          = "${vsphere_distributed_port_group.pg.name}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
  distributed_virtual_switch_uuid = "${vsphere_distributed_virtual_switch.dvs.id}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
	)
}

func testAccDataSourceVSphereNetworkConfigDVSPortgroupAbsolute() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_distributed_virtual_switch" "dvs" {
  name          = "terraform-test-dvs"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_distributed_port_group" "pg" {
  name                            = "terraform-test-pg"
  distributed_virtual_switch_uuid = "${vsphere_distributed_virtual_switch.dvs.id}"
}

data "vsphere_network" "net" {
  name          = "/${var.datacenter}/network/${vsphere_distributed_port_group.pg.name}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
	)
}

func testAccDataSourceVSphereNetworkConfigHostPortgroup() string {
	return fmt.Sprintf(`
data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

resource "vsphere_host_virtual_switch" "switch" {
  name           = "vSwitchTerraformTest"
  host_system_id = "${data.vsphere_host.esxi_host.id}"

  network_adapters = ["${var.host_nic0}", "${var.host_nic1}"]
  active_nics      = ["${var.host_nic0}", "${var.host_nic1}"]
  standby_nics     = []
}

resource "vsphere_host_port_group" "pg" {
  name                = "PGTerraformTest"
  host_system_id      = "${data.vsphere_host.esxi_host.id}"
  virtual_switch_name = "${vsphere_host_virtual_switch.switch.name}"
}

data "vsphere_network" "net" {
  name          = %s
  datacenter_id = data.vsphere_datacenter.datacenter.id
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("TF_VAR_VSPHERE_PG_NAME"),
	)
}

func testAccDataSourceVSphereNetworkConfigSimilarNet() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_network" "net" {
  name          = "/${var.datacenter}/network/%s"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("TF_VAR_VSPHERE_PG_NAME"),
	)
}
