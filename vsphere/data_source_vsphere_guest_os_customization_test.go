package vsphere

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestAccDataSourceVSphereGOSC_basic(t *testing.T) {
	goscName := acctest.RandomWithPrefix("lin")
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccDataSourceVSphereHostPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereGOSCConfig(goscName),
				Check:  resource.TestCheckResourceAttr("data.vsphere_guest_os_customization.gosc1", "id", goscName),
			},
		},
	})
}

func testAccDataSourceVSphereGOSCConfig(goscName string) string {
	return fmt.Sprintf(`
resource "vsphere_guest_os_customization" "source" {
			name = %q
			type = "Linux"
			spec {
				linux_options {
					domain = "example.com"
					host_name = "linux"
				}
			}
		}
data "vsphere_guest_os_customization" "gosc1" {
  name          = vsphere_guest_os_customization.source.id
}
`, goscName)
}
