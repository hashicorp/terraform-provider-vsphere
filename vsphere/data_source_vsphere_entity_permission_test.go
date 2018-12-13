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
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereEntityPermissionConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.vsphere_entity_permission.terraform-test-entity-permission-data",
						"principal",
						testAccDataSourceVSphereEntityPermissionCheckUser,
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

const testAccDataSourceVSphereEntityPermissionUser = os.Getenv("VSPHERE_USER")
const testAccDataSourceVSphereEntityPermissionCheckUser = os.Getenv("VSPHERE_CHECKUSER")

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
		testAccDataSourceVSphereEntityPermissionUser,
	)
}
