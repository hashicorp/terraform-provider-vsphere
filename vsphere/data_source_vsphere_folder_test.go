package vsphere

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

var testAccDataSourceVSphereFolderExpectedRegexp = regexp.MustCompile("^group-v")

func TestAccDataSourceVSphereFolder_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereFolderPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereFolderConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.vsphere_folder.folder",
						"id",
						testAccDataSourceVSphereFolderExpectedRegexp,
					),
				),
			},
		},
	})
}

func testAccDataSourceVSphereFolderPreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_FOLDER_V0_PATH") == "" {
		t.Skip("set VSPHERE_FOLDER_V0_PATH to run vsphere_folder acceptance tests")
	}
	if os.Getenv("VSPHERE_DATACENTER") == "" {
		t.Skip("set VSPHERE_DATACENTER to run vsphere_folder acceptance tests")
	}
}

func testAccDataSourceVSphereFolderConfig() string {
	return fmt.Sprintf(`
data "vsphere_folder" "folder" {
  path = "/%s/vm/%s"
}
`, os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_FOLDER_V0_PATH"))
}

const testAccDataSourceVSphereFolderConfigDefault = `
data "vsphere_datacenter" "dc" {}
`
