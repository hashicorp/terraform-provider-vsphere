package vsphere

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccResourceVSphereEntityPermission_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereEntityPermissionExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereEntityPermissionConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereEntityPermissionExists(true),
					testAccResourceVSphereEntityPermissionPrincipal("VSPHERE.HASHICORPTEST.INTERNAL\\teamcity"),
					testAccResourceVSphereEntityPermissionPropagate(true),
					testAccResourceVSphereEntityPermissionGroup(false),
				),
			},
		},
	})
}

func testAccResourceVSphereEntityPermissionExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ep, err := testGetEntityPermission(s, "entity_permission")
		if err != nil && !strings.Contains(err.Error(), "no principal with name") {
			return err
		}
		if ep == nil && expected {
			return fmt.Errorf("entity permission %q is not found", ep.RoleId)
		}
		if ep != nil && !expected {
			return fmt.Errorf("expected entity_permission to be missing")
		}
		return nil
	}
}

func testAccResourceVSphereEntityPermissionPrincipal(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ep, err := testGetEntityPermission(s, "entity_permission")
		if err != nil {
			return err
		}
		if ep.Principal != expected {
			return fmt.Errorf("expected entity_permission principal to be %q, got %q", expected, ep.Principal)
		}
		return nil
	}
}

func testAccResourceVSphereEntityPermissionGroup(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ep, err := testGetEntityPermission(s, "entity_permission")
		if err != nil {
			return err
		}
		if ep.Group != expected {
			return fmt.Errorf("expected entity_permission group to be %t, got %t", expected, ep.Group)
		}
		return nil
	}
}

func testAccResourceVSphereEntityPermissionPropagate(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ep, err := testGetEntityPermission(s, "entity_permission")
		if err != nil {
			return err
		}
		if ep.Propagate != expected {
			return fmt.Errorf("expected entity_permission propagate to be %t, got %t", expected, ep.Propagate)
		}
		return nil
	}
}

func testAccResourceVSphereEntityPermissionConfigBasic() string {
	return fmt.Sprintf(`
variable "datacenter" {
	default = "%s"
}

data "vsphere_datacenter" "dc" {
	name = "${var.datacenter}"
}

data "vsphere_role" "default" {
	name = "tfTestPermission"
}

data "vsphere_datastore" "datastore" {
	name = "nfsds2"
	datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_entity_permission" "entity_permission" {
	principal   = "VSPHERE.HASHICORPTEST.INTERNAL\\teamcity"
	role_id     = "${data.vsphere_role.default.id}"
	entity_id   = "${data.vsphere_datastore.datastore.id}"
	entity_type = "Datastore"
	propagate   = true
}
`,
		"hashi-dc",
	)
}
