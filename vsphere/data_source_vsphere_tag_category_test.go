package vsphere

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceVSphereTagCategory_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereTagCategoryConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.vsphere_tag_category.terraform-test-category-data",
						"name",
						testAccDataSourceVSphereTagCategoryConfigName,
					),
					resource.TestCheckResourceAttr(
						"data.vsphere_tag_category.terraform-test-category-data",
						"description",
						testAccDataSourceVSphereTagCategoryConfigDescription,
					),
					resource.TestCheckResourceAttr(
						"data.vsphere_tag_category.terraform-test-category-data",
						"cardinality",
						testAccDataSourceVSphereTagCategoryConfigCardinality,
					),
					resource.TestCheckResourceAttr(
						"data.vsphere_tag_category.terraform-test-category-data",
						"associable_types.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"data.vsphere_tag_category.terraform-test-category-data",
						"associable_types.3125094965",
						testAccDataSourceVSphereTagCategoryConfigAssociableType,
					),
					resource.TestCheckResourceAttrPair(
						"data.vsphere_tag_category.terraform-test-category-data", "id",
						"vsphere_tag_category.terraform-test-category", "id",
					),
				),
			},
		},
	})
}

const testAccDataSourceVSphereTagCategoryConfigName = "terraform-test-category"
const testAccDataSourceVSphereTagCategoryConfigDescription = "Managed by Terraform"
const testAccDataSourceVSphereTagCategoryConfigCardinality = vSphereTagCategoryCardinalitySingle
const testAccDataSourceVSphereTagCategoryConfigAssociableType = vSphereTagTypeVirtualMachine

func testAccDataSourceVSphereTagCategoryConfig() string {
	return fmt.Sprintf(`
variable "tag_category_name" {
  default = "%s"
}

variable "tag_category_description" {
  default = "%s"
}

variable "tag_category_cardinality" {
  default = "%s"
}

variable "tag_category_associable_types" {
  default = [
    "%s",
  ]
}

resource "vsphere_tag_category" "terraform-test-category" {
  name        = "${var.tag_category_name}"
  description = "${var.tag_category_description}"
  cardinality = "${var.tag_category_cardinality}"

  associable_types = [
    "${var.tag_category_associable_types}",
  ]
}

data "vsphere_tag_category" "terraform-test-category-data" {
  name = "${vsphere_tag_category.terraform-test-category.name}"
}
`,
		testAccDataSourceVSphereTagCategoryConfigName,
		testAccDataSourceVSphereTagCategoryConfigDescription,
		testAccDataSourceVSphereTagCategoryConfigCardinality,
		testAccDataSourceVSphereTagCategoryConfigAssociableType,
	)
}
