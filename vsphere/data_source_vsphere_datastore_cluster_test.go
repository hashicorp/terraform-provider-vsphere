package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceVSphereDatastoreCluster_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDatastoreClusterPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDatastoreClusterConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.vsphere_datastore_cluster.datastore_cluster_data", "id",
						"vsphere_datastore_cluster.datastore_cluster", "id",
					),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereDatastoreCluster_absolutePathNoDatacenter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDatastoreClusterPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDatastoreClusterConfigAbsolutePath(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.vsphere_datastore_cluster.datastore_cluster_data", "id",
						"vsphere_datastore_cluster.datastore_cluster", "id",
					),
				),
			},
		},
	})
}

func testAccDataSourceVSphereDatastoreClusterConfigBasic() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_datastore_cluster" "datastore_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_datastore_cluster" "datastore_cluster_data" {
  name          = "${vsphere_datastore_cluster.datastore_cluster.name}"
  datacenter_id = "${vsphere_datastore_cluster.datastore_cluster.datacenter_id}"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
	)
}

func testAccDataSourceVSphereDatastoreClusterConfigAbsolutePath() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_datastore_cluster" "datastore_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_datastore_cluster" "datastore_cluster_data" {
  name          = "/${var.datacenter}/datastore/${vsphere_datastore_cluster.datastore_cluster.name}"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
	)
}
