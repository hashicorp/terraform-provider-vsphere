package vsphere

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceVSphereCustomAttribute_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereCustomAttributeConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.vsphere_custom_attribute.terraform-test-attribute-data",
						"name",
						testAccDataSourceVSphereCustomAttributeConfigName,
					),
					resource.TestCheckResourceAttr(
						"data.vsphere_custom_attribute.terraform-test-attribute-data",
						"managed_object_type",
						testAccDataSourceVSphereCustomAttributeConfigType,
					),
					resource.TestCheckResourceAttrPair(
						"data.vsphere_custom_attribute.terraform-test-attribute-data", "id",
						"vsphere_custom_attribute.terraform-test-attribute", "id",
					),
				),
			},
		},
	})
}

const testAccDataSourceVSphereCustomAttributeConfigName = "terraform-test-attribute"
const testAccDataSourceVSphereCustomAttributeConfigType = "VirtualMachine"

func testAccDataSourceVSphereCustomAttributeConfig() string {
	return fmt.Sprintf(`
variable "attribute_name" {
  default = "%s"
}

variable "attribute_type" {
  default = "%s"
}

resource "vsphere_custom_attribute" "terraform-test-attribute" {
  name                = "${var.attribute_name}"
  managed_object_type = "${var.attribute_type}"
}

data "vsphere_custom_attribute" "terraform-test-attribute-data" {
  name = "${vsphere_custom_attribute.terraform-test-attribute.name}"
}
`,
		testAccDataSourceVSphereCustomAttributeConfigName,
		testAccDataSourceVSphereCustomAttributeConfigType,
	)
}
