package vsphere

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// The Administrator user is created by default on vCenter Server systems.
const SsoUserName = "Administrator"
const SsoUserGeneratedId = "Administrator.vsphere.local"

func TestAccDataSourceVSphereSsoUser_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			// RunSweepers()
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereSsoUserConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.vsphere_sso_user.user", "name", SsoUserName),
					resource.TestCheckResourceAttr("data.vsphere_sso_user.user", "id", SsoUserGeneratedId),
				),
			},
		},
	})
}

func testAccDataSourceVSphereSsoUserConfig() string {
	return fmt.Sprintf(`
data "vsphere_sso_user" "user" {
  name = "%s"
}
`,
		SsoUserName,
	)
}
