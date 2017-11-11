package vsphere

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceVSphereVirtualMachine(t *testing.T) {
	var tp *testing.T
	testAccDataSourceVSphereVirtualMachineCases := []struct {
		name     string
		testCase resource.TestCase
	}{
		{
			"basic",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccDataSourceVSphereVirtualMachinePreCheck(tp)
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
						),
					},
				},
			},
		},
		{
			"no datacenter and absolute path",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccDataSourceVSphereVirtualMachinePreCheck(tp)
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
						),
					},
				},
			},
		},
	}

	for _, tc := range testAccDataSourceVSphereVirtualMachineCases {
		t.Run(tc.name, func(t *testing.T) {
			tp = t
			resource.Test(t, tc.testCase)
		})
	}
}

func testAccDataSourceVSphereVirtualMachinePreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_DATACENTER") == "" {
		t.Skip("set VSPHERE_DATACENTER to run vsphere_virtual_machine data source acceptance tests")
	}
	if os.Getenv("VSPHERE_TEMPLATE") == "" {
		t.Skip("set VSPHERE_TEMPLATE to run vsphere_virtual_machine data source acceptance tests")
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
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_TEMPLATE"),
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
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_TEMPLATE"),
	)
}
