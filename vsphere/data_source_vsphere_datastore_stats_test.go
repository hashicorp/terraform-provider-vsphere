// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceVSphereDatastoreStats_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccDataSourceVSphereDatastoreStatsPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDatastoreStatsConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.vsphere_datastore_stats.datastore_stats", "datacenter_id", os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
					),
					resource.TestCheckResourceAttr(
						"data.vsphere_datastore_stats.datastore_stats", "id", fmt.Sprintf("%s_stats", os.Getenv("TF_VAR_VSPHERE_DATACENTER")),
					),
					testCheckOutputBool("found_free_space", "true"),
					testCheckOutputBool("found_capacity", "true"),
					testCheckOutputBool("free_values_exist", "true"),
					testCheckOutputBool("capacity_values_exist", "true"),
				),
			},
		},
	})
}

func testAccDataSourceVSphereDatastoreStatsPreCheck(t *testing.T) {
	if os.Getenv("TF_VAR_VSPHERE_DATACENTER") == "" {
		t.Skip("set TF_VAR_VSPHERE_DATACENTER to run vsphere_datastore_stats acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_USER") == "" {
		t.Skip("set TF_VAR_VSPHERE_USER to run vsphere_datastore_stats acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_PASSWORD") == "" {
		t.Skip("set TF_VAR_VSPHERE_PASSWORD to run vsphere_datastore_stats acceptance tests")
	}
}

func testAccDataSourceVSphereDatastoreStatsConfig() string {
	return fmt.Sprintf(`
variable "datacenter_id" {
	default = "%s"
}

data "vsphere_datastore_stats" "datastore_stats" {
  datacenter_id = "${var.datacenter_id}"
}

output "found_free_space" {
	value = "${length(data.vsphere_datastore_stats.datastore_stats.free_space) >= 1 ? "true" : "false" }"
}

output "found_capacity" {
	value = "${length(data.vsphere_datastore_stats.datastore_stats.capacity) >= 1 ? "true" : "false" }"
}

output "free_values_exist" {
	value = alltrue([
		for free in values(data.vsphere_datastore_stats.datastore_stats.free_space):
		free >= 1
	])
}

output "capacity_values_exist" {
	value = alltrue([
		for free in values(data.vsphere_datastore_stats.datastore_stats.capacity):
		free >= 1
	])
}
`, os.Getenv("TF_VAR_VSPHERE_DATACENTER"))
}
