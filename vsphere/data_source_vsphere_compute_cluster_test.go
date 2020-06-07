package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccDataSourceVSphereComputeCluster_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereComputeClusterConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.vsphere_compute_cluster.compute_cluster_data", "id",
						"vsphere_compute_cluster.compute_cluster", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.vsphere_compute_cluster.compute_cluster_data", "resource_pool_id",
						"vsphere_compute_cluster.compute_cluster", "resource_pool_id",
					),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereComputeCluster_absolutePathNoDatacenter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereComputeClusterConfigAbsolutePath(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.vsphere_compute_cluster.compute_cluster_data", "id",
						"vsphere_compute_cluster.compute_cluster", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.vsphere_compute_cluster.compute_cluster_data", "resource_pool_id",
						"vsphere_compute_cluster.compute_cluster", "resource_pool_id",
					),
				),
			},
		},
	})
}

func testAccDataSourceVSphereComputeClusterConfigBasic() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_compute_cluster" "compute_cluster_data" {
  name          = "${vsphere_compute_cluster.compute_cluster.name}"
  datacenter_id = "${vsphere_compute_cluster.compute_cluster.datacenter_id}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
	)
}

func testAccDataSourceVSphereComputeClusterConfigAbsolutePath() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_compute_cluster" "compute_cluster_data" {
  name          = "/${var.datacenter}/host/${vsphere_compute_cluster.compute_cluster.name}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
	)
}
