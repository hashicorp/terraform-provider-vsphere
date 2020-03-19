package vsphere

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccDataSourceVSphereContentLibraryItem_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereContentLibraryItemPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereContentLibraryItemConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.vsphere_content_library_item.item", "id", regexp.MustCompile("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$"),
					),
				),
			},
		},
	})
}

func testAccDataSourceVSphereContentLibraryItemPreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_CONTENT_LIBRARY") == "" {
		t.Skip("set VSPHERE_CONTENT_LIBRARY to run vsphere_content_library acceptance tests")
	}
	if os.Getenv("VSPHERE_CONTENT_LIBRARY_ITEM") == "" {
		t.Skip("set VSPHERE_CONTENT_LIBRARY_ITEM to run vsphere_content_library_item acceptance tests")
	}
}

func testAccDataSourceVSphereContentLibraryItemConfig() string {
	return fmt.Sprintf(`
variable "content_library" {
  type    = "string"
  default = "%s"
}

variable "content_library_item" {
  type    = "string"
  default = "%s"
}

data "vsphere_content_library" "library" {
  name = var.content_library
}

data "vsphere_content_library_item" "item" {
  name       = var.content_library_item
  library_id = data.vsphere_content_library.library.id
}
`,
		os.Getenv("VSPHERE_CONTENT_LIBRARY"),
		os.Getenv("VSPHERE_CONTENT_LIBRARY_ITEM"),
	)
}
