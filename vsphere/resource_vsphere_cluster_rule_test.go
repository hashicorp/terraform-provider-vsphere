package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/clusterrule"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/computeresource"
)

func TestAccResourceVSphereClusterRule(t *testing.T) {
	var tp *testing.T
	testAccResourceVSphereClusterRuleCases := []struct {
		name     string
		testCase resource.TestCase
	}{
		{
			"affinity",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
				},
				Providers: testAccProviders,
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereClusterRuleAffinityConfigBasic(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereClusterRuleExists(),
						),
					},
					{
						Config: testAccResourceVSphereClusterRuleAffinityConfigUpdate(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereClusterRuleExists(),
						),
					},
				},
			},
		},
		{
			"antiaffinity",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
				},
				Providers: testAccProviders,
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereClusterRuleAntiAffinityConfigBasic(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereClusterRuleExists(),
						),
					},
				},
			},
		},
		{
			"hostaffine",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
				},
				Providers: testAccProviders,
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereClusterRuleHostAffineConfigBasic(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereClusterRuleExists(),
						),
					},
				},
			},
		},
		{
			"hostantiaffine",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
				},
				Providers: testAccProviders,
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereClusterRuleHostAntiAffineConfigBasic(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereClusterRuleExists(),
						),
					},
				},
			},
		},
	}

	for _, tc := range testAccResourceVSphereClusterRuleCases {
		t.Run(tc.name, func(t *testing.T) {
			tp = t
			resource.Test(t, tc.testCase)
		})
	}
}

func testAccResourceVSphereClusterRuleAffinityConfigBasic() string {
	return fmt.Sprintf(`

resource "vsphere_cluster_rule" "foo" {
  name = "terraform-cluster-rule"
  type = "affinity"

  datacenter_id               = "%s"
  cluster_compute_resource_id = "%s"

  virtual_machines = [
    "%s",
  ]
}
`, os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_CLUSTER"), os.Getenv("VSPHERE_VM_V1_PATH"))
}

func testAccResourceVSphereClusterRuleAffinityConfigUpdate() string {
	return fmt.Sprintf(`

resource "vsphere_cluster_rule" "foo" {
  name = "terraform-cluster-rule"
  type = "affinity"

  datacenter_id               = "%s"
  cluster_compute_resource_id = "%s"

  virtual_machines = [
    "%s",
  ]
}
`, os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_CLUSTER"), os.Getenv("VSPHERE_VM_V2_PATH"))
}

func testAccResourceVSphereClusterRuleAntiAffinityConfigBasic() string {
	return fmt.Sprintf(`

resource "vsphere_cluster_rule" "foo" {
  name = "terraform-cluster-rule"
  type = "antiaffinity"

  datacenter_id               = "%s"
  cluster_compute_resource_id = "%s"

  virtual_machines = [
    "%s",
  ]

}
`, os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_CLUSTER"), os.Getenv("VSPHERE_VM_V1_PATH"))
}

func testAccResourceVSphereClusterRuleHostAffineConfigBasic() string {
	return fmt.Sprintf(`

resource "vsphere_cluster_rule" "foo" {
  name = "terraform-cluster-rule"
  type = "vmhostaffine"

  datacenter_id               = "%s"
  cluster_compute_resource_id = "%s"

  vm_group_name   = "%s"
  host_group_name = "%s"
}
`, os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_CLUSTER"), os.Getenv("VSPHERE_VM_GROUP_NAME"), os.Getenv("VSPHERE_HOST_GROUP_NAME"))
}

func testAccResourceVSphereClusterRuleHostAntiAffineConfigBasic() string {
	return fmt.Sprintf(`

resource "vsphere_cluster_rule" "foo" {
  name = "terraform-cluster-rule"
  type = "vmhostantiaffine"

  datacenter_id               = "%s"
  cluster_compute_resource_id = "%s"

  vm_group_name   = "%s"
  host_group_name = "%s"
}
`, os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_CLUSTER"), os.Getenv("VSPHERE_VM_GROUP_NAME"), os.Getenv("VSPHERE_HOST_GROUP_NAME"))
}

func testAccResourceVSphereClusterRuleExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		name := "vsphere_cluster_rule.foo"
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		fmt.Printf("%v\n", rs.Primary.Attributes)
		client := testAccProvider.Meta().(*VSphereClient).vimClient
		ccr, err := computeresource.ClusterFromID(client, rs.Primary.Attributes["cluster_compute_resource_id"])
		if err != nil {
			return err
		}

		_, err = clusterrule.GetRuleByName(ccr, rs.Primary.Attributes["name"])
		if err != nil {
			return err
		}

		return nil
	}
}
