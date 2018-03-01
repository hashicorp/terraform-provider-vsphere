package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceVSphereDatastore_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereDatastorePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDatastoreConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.vsphere_datastore.datastore_data", "id",
						"vsphere_nas_datastore.datastore", "id",
					),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereDatastore_noDatacenterAndAbsolutePath(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereDatastorePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDatastoreConfigAbsolutePath(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.vsphere_datastore.datastore_data", "id",
						"vsphere_nas_datastore.datastore", "id",
					),
				),
			},
		},
	})
}

func testAccDataSourceVSphereDatastorePreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_ESXI_HOST") == "" {
		t.Skip("set VSPHERE_ESXI_HOST to run vsphere_vmfs_disks acceptance tests")
	}
	if os.Getenv("VSPHERE_NAS_HOST") == "" {
		t.Skip("set VSPHERE_NAS_HOST to run vsphere_nas_datastore acceptance tests")
	}
	if os.Getenv("VSPHERE_NFS_PATH") == "" {
		t.Skip("set VSPHERE_NFS_PATH to run vsphere_nas_datastore acceptance tests")
	}
	if os.Getenv("VSPHERE_DS_FOLDER") == "" {
		t.Skip("set VSPHERE_DS_FOLDER to run vsphere_nas_datastore acceptance tests")
	}
}

func testAccDataSourceVSphereDatastoreConfig() string {
	return fmt.Sprintf(`
variable "nfs_host" {
  type    = "string"
  default = "%s"
}

variable "nfs_path" {
  type    = "string"
  default = "%s"
}

data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_host" "esxi_host" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_nas_datastore" "datastore" {
  name            = "terraform-test-nas"
  host_system_ids = ["${data.vsphere_host.esxi_host.id}"]

  type         = "NFS"
  remote_hosts = ["${var.nfs_host}"]
  remote_path  = "${var.nfs_path}"
}

data "vsphere_datastore" "datastore_data" {
  name          = "${vsphere_nas_datastore.datastore.name}"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}
`,
		os.Getenv("VSPHERE_NAS_HOST"),
		os.Getenv("VSPHERE_NFS_PATH"),
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_ESXI_HOST"),
	)
}

func testAccDataSourceVSphereDatastoreConfigAbsolutePath() string {
	return fmt.Sprintf(`
variable "nfs_host" {
  type    = "string"
  default = "%s"
}

variable "nfs_path" {
  type    = "string"
  default = "%s"
}

data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_host" "esxi_host" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_nas_datastore" "datastore" {
  name            = "terraform-test-nas"
  host_system_ids = ["${data.vsphere_host.esxi_host.id}"]

  type         = "NFS"
  remote_hosts = ["${var.nfs_host}"]
  remote_path  = "${var.nfs_path}"
}

data "vsphere_datastore" "datastore_data" {
  name = "/${data.vsphere_datacenter.datacenter.name}/datastore/${vsphere_nas_datastore.datastore.name}"
}
`,
		os.Getenv("VSPHERE_NAS_HOST"),
		os.Getenv("VSPHERE_NFS_PATH"),
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_ESXI_HOST"),
	)
}
