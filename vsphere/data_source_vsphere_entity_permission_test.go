package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceVSphereEntityPermission_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereEntityPermissionPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereEntityPermissionConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.vsphere_entity_permission.terraform-test-entity-permission-data",
						"principal",
						os.Getenv("VSPHERE_USER"),
					),
					resource.TestCheckResourceAttrPair(
						"data.vsphere_entity_permission.terraform-test-entity-permission-data", "principal",
						"vsphere_entity_permission.terraform-test-entity-permission", "principal",
					),
				),
			},
		},
	})
}

func testAccDataSourceVSphereEntityPermissionConfig() string {
	return fmt.Sprintf(`
variable "vsphere_user" {
  default = "%s"
}

resource "vsphere_entity_permission" "terraform-test-entity-permission" {
  principal   = "${var.vsphere_user}"
  role_id     = -1
  folder_path = "/"
}

data "vsphere_entity_permission" "terraform-test-entity-permission-data" {
  principal = "${var.vsphere_user}"
}
`,
		os.Getenv("VSPHERE_USER"),
	)
}

func testAccDataSourceVSphereEntityPermissionPreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_USER") == "" {
		t.Skip("set VSPHERE_USER to run vsphere_entity_permission acceptance tests")
	}
}
