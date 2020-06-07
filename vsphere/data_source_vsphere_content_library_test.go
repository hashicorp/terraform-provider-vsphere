package vsphere

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccDataSourceVSphereContentLibrary_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereContentLibraryPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereContentLibraryConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.vsphere_content_library.library", "id", regexp.MustCompile("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$"),
					),
				),
			},
		},
	})
}

func testAccDataSourceVSphereContentLibraryPreCheck(t *testing.T) {
	if os.Getenv("TF_VAR_VSPHERE_CONTENT_LIBRARY_FILES") == "" {
		t.Skip("set TF_VAR_VSPHERE_CONTENT_LIBRARY_FILES to run vsphere_content_library acceptance tests")
	}
}

func testAccDataSourceVSphereContentLibraryConfig() string {
	return fmt.Sprintf(`
variable "datacenter" {
  type    = "string"
  default = "%s"
}

variable "datastore" {
  type    = "string"
  default = "%s"
}

variable "file_list" {
  type    = list(string)
  default = %s 
}

data "vsphere_datacenter" "dc" {
  name = var.datacenter
}

data "vsphere_datastore" "ds" {
  datacenter_id = data.vsphere_datacenter.dc.id
  name = var.datastore
}

resource "vsphere_content_library" "library" {
  name            = "ContentLibrary_test"
  storage_backing = [ data.vsphere_datastore.ds.id ]
  description     = "Library Description"
}

data "vsphere_content_library" "library" {
  name = vsphere_content_library.library.name
}

`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"),
		os.Getenv("TF_VAR_VSPHERE_CONTENT_LIBRARY_FILES"),
	)
}
