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
	"github.com/vmware/terraform-provider-vsphere/vsphere/internal/helper/testhelper"
)

func TestAccDataSourceVSphereResourcePool_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccSkipIfEsxi(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereResourcePoolConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.vsphere_resource_pool.pool", "id", regexp.MustCompile("^resgroup-")),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereResourcePool_noDatacenterAndAbsolutePath(t *testing.T) {
	testAccSkipUnstable(t)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccDataSourceVSphereResourcePoolPreCheck(t)
			testAccSkipIfEsxi(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereResourcePoolConfigAbsolutePath(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.vsphere_resource_pool.pool", "id", regexp.MustCompile("^resgroup-")),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereResourcePool_withParentId(t *testing.T) {
	testAccSkipUnstable(t)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccDataSourceVSphereResourcePoolPreCheck(t)
			testAccSkipIfEsxi(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereResourcePoolConfigWithParent(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.vsphere_resource_pool.pool_by_parent", "id"),
					resource.TestMatchResourceAttr("data.vsphere_resource_pool.pool_by_parent", "id", regexp.MustCompile("^resgroup-")),
					resource.TestCheckResourceAttrPair(
						"data.vsphere_resource_pool.pool_by_parent", "id",
						"data.vsphere_resource_pool.parent_pool", "id",
					),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereResourcePool_withParentIdAndNamePathError(t *testing.T) {
	testAccSkipUnstable(t)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccDataSourceVSphereResourcePoolPreCheck(t)
			testAccSkipIfEsxi(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceVSphereResourcePoolConfigWithParentAndNamePath(),
				ExpectError: regexp.MustCompile("argument 'name' cannot be a path when 'parent_resource_pool_id' is specified"),
				PlanOnly:    true,
			},
		},
	})
}

func TestAccDataSourceVSphereResourcePool_withParentIdAndMissingNameError(t *testing.T) {
	testAccSkipUnstable(t)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccDataSourceVSphereResourcePoolPreCheck(t)
			testAccSkipIfEsxi(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceVSphereResourcePoolConfigWithParentAndMissingName(),
				ExpectError: regexp.MustCompile("argument 'name' is required when 'parent_resource_pool_id' is specified"),
				PlanOnly:    true,
			},
		},
	})
}

func TestAccDataSourceVSphereResourcePool_withInvalidParentIdError(t *testing.T) {
	testAccSkipUnstable(t)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccDataSourceVSphereResourcePoolPreCheck(t)
			testAccSkipIfEsxi(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceVSphereResourcePoolConfigWithInvalidParent(),
				ExpectError: regexp.MustCompile("could not find parent resource pool"),
			},
		},
	})
}

func TestAccDataSourceVSphereResourcePool_withParentIdAndNotFoundNameError(t *testing.T) {
	testAccSkipUnstable(t)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccDataSourceVSphereResourcePoolPreCheck(t)
			testAccSkipIfEsxi(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceVSphereResourcePoolConfigWithParentAndNotFoundName(),
				ExpectError: regexp.MustCompile("resource pool .* not found under parent resource pool"),
			},
		},
	})
}

func TestAccDataSourceVSphereResourcePool_defaultResourcePoolForESXi(t *testing.T) {
	testAccSkipUnstable(t)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccDataSourceVSphereResourcePoolPreCheck(t)
			testAccSkipIfNotEsxi(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereResourcePoolConfigDefault,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.vsphere_resource_pool.pool", "id", regexp.MustCompile("^ha-root-pool$")),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereResourcePool_emptyNameOnVCenterShouldError(t *testing.T) {
	testAccSkipUnstable(t)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccDataSourceVSphereResourcePoolPreCheck(t)
			testAccSkipIfEsxi(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceVSphereResourcePoolConfigDefault,
				ExpectError: regexp.MustCompile("argument 'name' is required when 'parent_resource_pool_id' is not specified"), // Adjusted error message check
				PlanOnly:    true,
			},
		},
	})
}

func testAccDataSourceVSphereResourcePoolPreCheck(t *testing.T) {
	if os.Getenv("TF_VAR_VSPHERE_DATACENTER") == "" {
		t.Skip("set TF_VAR_VSPHERE_DATACENTER to run vsphere_resource_pool data source acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_CLUSTER") == "" {
		t.Skip("set TF_VAR_VSPHERE_CLUSTER to run vsphere_resource_pool data source acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_RESOURCE_POOL") == "" {
		t.Skip("set TF_VAR_VSPHERE_RESOURCE_POOL to run vsphere_resource_pool data source acceptance tests (must be a child pool of the cluster's default pool)")
	}
}

func testAccDataSourceVSphereResourcePoolConfig() string {
	return fmt.Sprintf(`
%s

resource "vsphere_resource_pool" "resource_pool" {
  name                    = "terraform-test-resource-pool"
  parent_resource_pool_id = data.vsphere_compute_cluster.rootcompute_cluster1.resource_pool_id
}

data "vsphere_resource_pool" "pool" {
  name          = vsphere_resource_pool.resource_pool.name
  datacenter_id = data.vsphere_datacenter.rootdc1.id
}
`,
		testhelper.CombineConfigs(testhelper.ConfigDataRootDC1(), testhelper.ConfigDataRootComputeCluster1()),
	)
}

func testAccDataSourceVSphereResourcePoolConfigAbsolutePath() string {
	return fmt.Sprintf(`
%s

variable "resource_pool_name" {
  description = "The name of the child resource pool to find (relative to cluster Resources)"
  default     = "%s"
}

data "vsphere_resource_pool" "pool" {
  name = "/${data.vsphere_datacenter.rootdc1.name}/host/${data.vsphere_compute_cluster.rootcompute_cluster1.name}/Resources/${var.resource_pool_name}"
}
`,
		testhelper.CombineConfigs(testhelper.ConfigDataRootDC1(), testhelper.ConfigDataRootComputeCluster1()),
		os.Getenv("TF_VAR_VSPHERE_RESOURCE_POOL"),
	)
}

func testAccDataSourceVSphereResourcePoolConfigWithParent() string {
	return fmt.Sprintf(`
%s

variable "resource_pool_name" {
  description = "The name of the child resource pool to find (relative to cluster Resources)"
  default     = "%s"
}

data "vsphere_resource_pool" "parent_pool" {
  name          = data.vsphere_compute_cluster.rootcompute_cluster1.name + "/Resources"
  datacenter_id = data.vsphere_datacenter.rootdc1.id
}

data "vsphere_resource_pool" "pool_by_parent" {
  name                    = var.resource_pool_name
  parent_resource_pool_id = data.vsphere_resource_pool.parent_pool.id
  datacenter_id           = data.vsphere_datacenter.rootdc1.id // Optional, but good practice
}
`,
		testhelper.CombineConfigs(testhelper.ConfigDataRootDC1(), testhelper.ConfigDataRootComputeCluster1()),
		os.Getenv("TF_VAR_VSPHERE_RESOURCE_POOL"),
	)
}

func testAccDataSourceVSphereResourcePoolConfigWithParentAndNamePath() string {
	return fmt.Sprintf(`
%s

data "vsphere_resource_pool" "parent_pool" {
  name          = data.vsphere_compute_cluster.rootcompute_cluster1.name + "/Resources"
  datacenter_id = data.vsphere_datacenter.rootdc1.id
}

data "vsphere_resource_pool" "pool_by_parent" {
  name                    = "${data.vsphere_compute_cluster.rootcompute_cluster1.name}/Resources/some_pool" // Path name
  parent_resource_pool_id = data.vsphere_resource_pool.parent_pool.id
  datacenter_id           = data.vsphere_datacenter.rootdc1.id
}
`,
		testhelper.CombineConfigs(testhelper.ConfigDataRootDC1(), testhelper.ConfigDataRootComputeCluster1()),
	)
}

func testAccDataSourceVSphereResourcePoolConfigWithParentAndMissingName() string {
	return fmt.Sprintf(`
%s

data "vsphere_resource_pool" "parent_pool" {
  name          = data.vsphere_compute_cluster.rootcompute_cluster1.name + "/Resources"
  datacenter_id = data.vsphere_datacenter.rootdc1.id
}

data "vsphere_resource_pool" "pool_by_parent" {
  parent_resource_pool_id = data.vsphere_resource_pool.parent_pool.id
  datacenter_id           = data.vsphere_datacenter.rootdc1.id
}
`,
		testhelper.CombineConfigs(testhelper.ConfigDataRootDC1(), testhelper.ConfigDataRootComputeCluster1()),
	)
}

func testAccDataSourceVSphereResourcePoolConfigWithInvalidParent() string {
	return fmt.Sprintf(`
%s

variable "resource_pool_name" {
  description = "The name of the child resource pool to find (relative to cluster Resources)"
  default     = "%s"
}

data "vsphere_resource_pool" "pool_by_parent" {
  name                    = var.resource_pool_name
  parent_resource_pool_id = "resgroup-000" // Invalid ID
  datacenter_id           = data.vsphere_datacenter.rootdc1.id
}
`,
		testhelper.CombineConfigs(testhelper.ConfigDataRootDC1(), testhelper.ConfigDataRootComputeCluster1()),
		os.Getenv("TF_VAR_VSPHERE_RESOURCE_POOL"),
	)
}

func testAccDataSourceVSphereResourcePoolConfigWithParentAndNotFoundName() string {
	return fmt.Sprintf(`
%s

data "vsphere_resource_pool" "parent_pool" {
  name          = data.vsphere_compute_cluster.rootcompute_cluster1.name + "/Resources"
  datacenter_id = data.vsphere_datacenter.rootdc1.id
}

data "vsphere_resource_pool" "pool_by_parent" {
  name                    = "non_existent_resource_pool_for_test"
  parent_resource_pool_id = data.vsphere_resource_pool.parent_pool.id
  datacenter_id           = data.vsphere_datacenter.rootdc1.id
}
`,
		testhelper.CombineConfigs(testhelper.ConfigDataRootDC1(), testhelper.ConfigDataRootComputeCluster1()),
	)
}

const testAccDataSourceVSphereResourcePoolConfigDefault = `
data "vsphere_resource_pool" "pool" {}
`
