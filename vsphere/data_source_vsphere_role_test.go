package vsphere

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceVSphereRole_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereRoleConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.vsphere_role.terraform-test-role-data",
						"id",
						"-1",
					),
					resource.TestCheckResourceAttr(
						"data.vsphere_role.terraform-test-role-data",
						"name",
						"Admin",
					),
				),
			},
		},
	})
}

func testAccDataSourceVSphereRoleConfig() string {
	return fmt.Sprintf(`
data "vsphere_role" "terraform-test-role-data" {
  role_id = -1
}
`,
	)
}
