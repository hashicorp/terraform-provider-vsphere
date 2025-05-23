// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testAccDataSourceVSphereDatacenterExpectedRegexp = regexp.MustCompile("^datacenter-")

func TestAccDataSourceVSphereDatacenter_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDatacenterConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.vsphere_datacenter.dc",
						"id",
						testAccDataSourceVSphereDatacenterExpectedRegexp,
					),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereDatacenter_defaultDatacenter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDatacenterConfigDefault,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.vsphere_datacenter.dc",
						"id",
						testAccDataSourceVSphereDatacenterExpectedRegexp,
					),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereDatacenter_getVirtualMachines(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDatacenterConfigGetVirtualMachines(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.vsphere_datacenter.dc",
						"id",
						testAccDataSourceVSphereDatacenterExpectedRegexp,
					),
					testCheckOutputBool("found_virtual_machines", "true"),
				),
			},
		},
	})
}

func testAccDataSourceVSphereDatacenterConfig() string {
	return fmt.Sprintf(`
data "vsphere_datacenter" "dc" {
  name = "%s"
}
`, os.Getenv("TF_VAR_VSPHERE_DATACENTER"))
}

const testAccDataSourceVSphereDatacenterConfigDefault = `
data "vsphere_datacenter" "dc" {}
`

func testAccDataSourceVSphereDatacenterConfigGetVirtualMachines() string {
	return fmt.Sprintf(`
data "vsphere_datacenter" "dc" {
  name = "%s"
}
output "found_virtual_machines" {
  value = length(data.vsphere_datacenter.dc.virtual_machines) >= 1 ? "true" : "false"
}
`, os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
	)
}
