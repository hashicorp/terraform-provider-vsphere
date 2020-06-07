package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
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
	if os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME") == "" {
		t.Skip("set TF_VAR_VSPHERE_NFS_DS_NAME to run vsphere_nas_datastore acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_DATACENTER") == "" {
		t.Skip("set TF_VAR_VSPHERE_DATACENTER to run vsphere_nas_datastore acceptance tests")
	}
}

func testAccDataSourceVSphereDatastoreConfig() string {
	return fmt.Sprintf(`
data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_datastore" "datastore_data" {
  name          = "%s"
  datacenter_id = data.vsphere_datacenter.datacenter.id
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"),
	)
}

func testAccDataSourceVSphereDatastoreConfigAbsolutePath() string {
	return fmt.Sprintf(`
data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_datastore" "datastore_data" {
  name = "/${data.vsphere_datacenter.datacenter.name}/datastore/%s"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"),
	)
}
