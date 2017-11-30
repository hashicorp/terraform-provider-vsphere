package vsphere

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceVSphereResourcePool(t *testing.T) {
	var tp *testing.T
	testAccDataSourceVSphereResourcePoolCases := []struct {
		name     string
		testCase resource.TestCase
	}{
		{
			"basic",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccDataSourceVSphereResourcePoolPreCheck(tp)
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
			},
		},
		{
			"no datacenter and absolute path",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccDataSourceVSphereResourcePoolPreCheck(tp)
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
			},
		},
	}

	for _, tc := range testAccDataSourceVSphereResourcePoolCases {
		t.Run(tc.name, func(t *testing.T) {
			tp = t
			resource.Test(t, tc.testCase)
		})
	}
}

func testAccDataSourceVSphereResourcePoolPreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_DATACENTER") == "" {
		t.Skip("set VSPHERE_DATACENTER to run vsphere_resource_pool data source acceptance tests")
	}
	if os.Getenv("VSPHERE_CLUSTER") == "" {
		t.Skip("set VSPHERE_CLUSTER to run vsphere_resource_pool data source acceptance tests")
	}
	if os.Getenv("VSPHERE_RESOURCE_POOL") == "" {
		t.Skip("set VSPHERE_RESOURCE_POOL to run vsphere_resource_pool data source acceptance tests")
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
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
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
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
	)
}
