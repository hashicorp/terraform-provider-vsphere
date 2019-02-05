package vsphere

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccResourceVSphereRole_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereRoleExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereRoleConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereRoleHasName("TestRole"),
					testAccResourceVSphereRoleHasPrivileges([]string{"System.Anonymous", "System.Read", "System.View", "VirtualMachine.State.CreateSnapshot"}),
					testAccResourceVSphereRoleExists(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereRole_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereRoleExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereRoleConfigUpdate(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereRoleExists(true),
					testAccResourceVSphereRoleHasName("TestRole2"),
					testAccResourceVSphereRoleHasPrivileges([]string{"System.Anonymous", "System.Read", "System.View", "Alarm.Create"}),
				),
			},
		},
	})
}

func testAccResourceVSphereRoleHasName(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		role, err := testGetRole(s, "role")
		if err != nil {
			return err
		}
		if role.Name != name {
			return fmt.Errorf("expected role to be named %q, got %q", name, role.Name)
		}
		return nil
	}
}

func testAccResourceVSphereRoleHasPrivileges(p []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		role, err := testGetRole(s, "role")
		if err != nil {
			return err
		}
		if len(p) != len(role.Privilege) {
			fmt.Errorf("expected role to have privileges %q, got %q", p, role.Privilege)
		}
		for _, pn := range role.Privilege {
			if !testAccResourceVSphereRoleHasPriv(p, pn) {
				return fmt.Errorf("expected role to have privileges %q, got %q", p, role.Privilege)
			}
		}
		return nil
	}
}

func testAccResourceVSphereRoleHasPriv(privileges []string, name string) bool {
	for _, p := range privileges {
		if p == name {
			return true
		}
	}
	return false
}

func testAccResourceVSphereRoleConfigBasic() string {
	return fmt.Sprintf(`
resource "vsphere_role" "role" {
	name        = "TestRole"
	permissions = ["VirtualMachine.State.CreateSnapshot"]
}
`,
	)
}

func testAccResourceVSphereRoleConfigUpdate() string {
	return fmt.Sprintf(`
resource "vsphere_role" "role" {
	name        = "TestRole2"
	permissions = ["Alarm.Create"]
}
`,
	)
}

func testAccResourceVSphereRoleExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		r, err := testGetRole(s, "role")
		if err != nil {
			return err
		}
		if r == nil {
			if expected == false {
				// Expected missing
				return nil
			}
			return fmt.Errorf("unable to find role")
		}
		if !expected {
			return fmt.Errorf("expected role %q to be missing", r.Name)
		}
		return nil
	}
}
