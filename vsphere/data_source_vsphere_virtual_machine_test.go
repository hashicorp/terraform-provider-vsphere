package vsphere

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccDataSourceVSphereVirtualMachine_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereVirtualMachineConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.vsphere_virtual_machine.template",
						"id",
						regexp.MustCompile("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$")),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "guest_id"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "scsi_type"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.#"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.0.size"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.0.eagerly_scrub"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.0.thin_provisioned"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interface_types.#"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "firmware"),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereVirtualMachine_noDatacenterAndAbsolutePath(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereVirtualMachineConfigAbsolutePath(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.vsphere_virtual_machine.template",
						"id",
						regexp.MustCompile("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$")),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "guest_id"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "scsi_type"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.#"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.0.size"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.0.eagerly_scrub"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.0.thin_provisioned"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interface_types.#"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "firmware"),
				),
			},
		},
	})
}

func testAccDataSourceVSphereVirtualMachinePreCheck(t *testing.T) {
	if os.Getenv("TF_VAR_VSPHERE_DATACENTER") == "" {
		t.Skip("set TF_VAR_VSPHERE_DATACENTER to run vsphere_virtual_machine data source acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_TEMPLATE") == "" {
		t.Skip("set TF_VAR_VSPHERE_TEMPLATE to run vsphere_virtual_machine data source acceptance tests")
	}
}

func testAccDataSourceVSphereVirtualMachineConfig() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("TF_VAR_VSPHERE_TEMPLATE"),
	)
}

func testAccDataSourceVSphereVirtualMachineConfigAbsolutePath() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

data "vsphere_virtual_machine" "template" {
  name = "/${var.datacenter}/vm/${var.template}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("TF_VAR_VSPHERE_TEMPLATE"),
	)
}
