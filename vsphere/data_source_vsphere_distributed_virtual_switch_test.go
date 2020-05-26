package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccDataSourceVSphereDistributedVirtualSwitch_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDistributedVirtualSwitchConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.vsphere_distributed_virtual_switch.dvs-data",
						"uplinks.#",
						"2",
					),
					resource.TestCheckResourceAttr(
						"data.vsphere_distributed_virtual_switch.dvs-data",
						"uplinks.0",
						"tfup1",
					),
					resource.TestCheckResourceAttr(
						"data.vsphere_distributed_virtual_switch.dvs-data",
						"uplinks.1",
						"tfup2",
					),
					resource.TestCheckResourceAttrPair(
						"data.vsphere_distributed_virtual_switch.dvs-data", "id",
						"vsphere_distributed_virtual_switch.dvs", "id",
					),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereDistributedVirtualSwitch_absolutePathNoDatacenterSpecified(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDistributedVirtualSwitchConfigAbsolute(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.vsphere_distributed_virtual_switch.dvs-data",
						"uplinks.#",
						"2",
					),
					resource.TestCheckResourceAttr(
						"data.vsphere_distributed_virtual_switch.dvs-data",
						"uplinks.0",
						"tfup1",
					),
					resource.TestCheckResourceAttr(
						"data.vsphere_distributed_virtual_switch.dvs-data",
						"uplinks.1",
						"tfup2",
					),
					resource.TestCheckResourceAttrPair(
						"data.vsphere_distributed_virtual_switch.dvs-data", "id",
						"vsphere_distributed_virtual_switch.dvs", "id",
					),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereDistributedVirtualSwitch_CreatePortgroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDistributedVirtualSwitchConfigWithPortgroup(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"vsphere_distributed_port_group.pg",
						"active_uplinks.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"vsphere_distributed_port_group.pg",
						"standby_uplinks.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"vsphere_distributed_port_group.pg",
						"active_uplinks.0",
						"tfup1",
					),
					resource.TestCheckResourceAttr(
						"vsphere_distributed_port_group.pg",
						"standby_uplinks.0",
						"tfup2",
					),
				),
			},
		},
	})
}

func testAccDataSourceVSphereDistributedVirtualSwitchConfig() string {
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
  uplinks       = ["tfup1", "tfup2"]
}

data "vsphere_distributed_virtual_switch" "dvs-data" {
  name          = "${vsphere_distributed_virtual_switch.dvs.name}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
	)
}

func testAccDataSourceVSphereDistributedVirtualSwitchConfigWithPortgroup() string {
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
  uplinks       = ["tfup1", "tfup2"]
}

data "vsphere_distributed_virtual_switch" "dvs-data" {
  name          = "${vsphere_distributed_virtual_switch.dvs.name}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_distributed_port_group" "pg" {
  name                            = "terraform-test-pg"
  distributed_virtual_switch_uuid = "${data.vsphere_distributed_virtual_switch.dvs-data.id}"

  active_uplinks  = ["${data.vsphere_distributed_virtual_switch.dvs-data.uplinks[0]}"]
  standby_uplinks = ["${data.vsphere_distributed_virtual_switch.dvs-data.uplinks[1]}"]
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
	)
}

func testAccDataSourceVSphereDistributedVirtualSwitchConfigAbsolute() string {
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
  uplinks       = ["tfup1", "tfup2"]
}

data "vsphere_distributed_virtual_switch" "dvs-data" {
  name          = "/${var.datacenter}/network/${vsphere_distributed_virtual_switch.dvs.name}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
	)
}
