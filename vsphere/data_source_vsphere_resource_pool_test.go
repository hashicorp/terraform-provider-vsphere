package vsphere

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccDataSourceVSphereResourcePool_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereResourcePoolPreCheck(t)
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
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
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

func TestAccDataSourceVSphereResourcePool_defaultResourcePoolForESXi(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
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
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereResourcePoolPreCheck(t)
			testAccSkipIfEsxi(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceVSphereResourcePoolConfigDefault,
				ExpectError: regexp.MustCompile("name cannot be empty when using vCenter"),
				PlanOnly:    true,
			},
			{
				Config: testAccResourceVSphereEmpty,
				Check:  resource.ComposeTestCheckFunc(),
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
		t.Skip("set TF_VAR_VSPHERE_RESOURCE_POOL to run vsphere_resource_pool data source acceptance tests")
	}
}

func testAccDataSourceVSphereResourcePoolConfig() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.cluster}/Resources/${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("TF_VAR_VSPHERE_CLUSTER"),
		os.Getenv("TF_VAR_VSPHERE_RESOURCE_POOL"),
	)
}

func testAccDataSourceVSphereResourcePoolConfigAbsolutePath() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

data "vsphere_resource_pool" "pool" {
  name = "/${var.datacenter}/host/${var.cluster}/Resources/${var.resource_pool}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("TF_VAR_VSPHERE_CLUSTER"),
		os.Getenv("TF_VAR_VSPHERE_RESOURCE_POOL"),
	)
}

const testAccDataSourceVSphereResourcePoolConfigDefault = `
data "vsphere_resource_pool" "pool" {}
`
