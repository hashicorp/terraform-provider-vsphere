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
	if os.Getenv("VSPHERE_CONTENT_LIBRARY") == "" {
		t.Skip("set VSPHERE_CONTENT_LIBRARY to run vsphere_content_library acceptance tests")
	}
}

func testAccDataSourceVSphereContentLibraryConfig() string {
	return fmt.Sprintf(`
variable "content_library" {
  type    = "string"
  default = "%s"
}

data "vsphere_content_library" "library" {
  name = var.content_library
}
`,
		os.Getenv("VSPHERE_CONTENT_LIBRARY"),
	)
}
