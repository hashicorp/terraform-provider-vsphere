package vsphere

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccDataSourceVSphereComputePolicy_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereComputePolicyConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						testAccCheckVSphereComputePolicyDataSourceName,
						"name",
						testAccDataSourceVSphereComputePolicyConfigName,
					),
					resource.TestCheckResourceAttr(
						testAccCheckVSphereComputePolicyDataSourceName,
						"description",
						testAccDataSourceVSphereComputePolicyConfigDescription,
					),
					resource.TestCheckResourceAttr(
						testAccCheckVSphereComputePolicyDataSourceName,
						"policy_type",
						computePolicyTypeVmHostAffinity,
					),
					resource.TestCheckResourceAttrPair(
						testAccCheckVSphereComputePolicyDataSourceName, "id",
						testAccCheckVSphereComputePolicyResourceName, "id",
					),
				),
			},
		},
	})
}

const testAccCheckVSphereComputePolicyDataSourceName = "data.vsphere_compute_policy.terraform_test_compute_policy_data"
const testAccDataSourceVSphereComputePolicyConfigName = "terraform-test-compute-policy"
const testAccDataSourceVSphereComputePolicyConfigDescription = "Managed by Terraform"

func testAccDataSourceVSphereComputePolicyConfig() string {
	return fmt.Sprintf(`
variable "compute_policy_name" {
	default = "%s"
}
resource "vsphere_tag_category" "test_category" {
	name = "terraform-test-compute-policy-tag-category"
	description = "description"
	cardinality = "MULTIPLE"
	associable_types = [
	  "HostSystem",
	  "VirtualMachine"
	]
}
resource "vsphere_tag" "terraform_test_tag" {
	name = "terraform-test-compute-policy-tag"
	category_id = "${vsphere_tag_category.test_category.id}"
}
resource "vsphere_compute_policy" "terraform_test_policy" {
	name = "${var.compute_policy_name}"
	description = "%s"
	vm_tag = "${vsphere_tag.terraform_test_tag.id}"
	host_tag = "${vsphere_tag.terraform_test_tag.id}"
	policy_type = "%s"
}
data "vsphere_compute_policy" "terraform_test_compute_policy_data" {
	name = "${vsphere_compute_policy.terraform_test_policy.name}"
}
`,
		testAccDataSourceVSphereComputePolicyConfigName,
		testAccDataSourceVSphereComputePolicyConfigDescription,
		computePolicyTypeVmHostAffinity,
	)
}
