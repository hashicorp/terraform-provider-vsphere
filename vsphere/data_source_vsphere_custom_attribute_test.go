package vsphere

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceVSphereCustomAttribute(t *testing.T) {
	var tp *testing.T
	testAccDataSourceVSphereCustomAttributeCases := []struct {
		name     string
		testCase resource.TestCase
	}{
		{
			"basic",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
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
			},
		},
	}

	for _, tc := range testAccDataSourceVSphereCustomAttributeCases {
		t.Run(tc.name, func(t *testing.T) {
			tp = t
			resource.Test(t, tc.testCase)
		})
	}
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
