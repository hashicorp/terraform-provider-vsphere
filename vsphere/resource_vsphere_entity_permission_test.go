package vsphere

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccResourceVSphereEntityPermission(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		//CheckDestroy: testAccResourceVSphereEntityPermissionExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereEntityPermissionConfigBasic(),
				Check:  resource.ComposeTestCheckFunc(
				//testAccResourceVSphereEntityPermissionExists(true),
				),
			},
		},
	})
}

func testAccResourceVSphereEntityPermissionExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ep, err := testGetEntityPermission(s, "entity_permission")
		if err != nil {
			return err
		}
		if ep == nil && expected {
			return fmt.Errorf("entity permission %q is not found", ep.RoleId)
		}
		if ep != nil && !expected {
			return fmt.Errorf("expected entity permission %q to be missing", ep.RoleId)
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

resource "vsphere_folder" "folder" {
	path = "terraform-test-folder"
	type = "vm"
	datacenter_id = "${data.vsphere_datacenter.dc}"
}

data "vsphere_role" "default" {
	name = "Admin"
}

resource "vsphere_entity_permission" "entity_permission" {
	principal = "VSPHERE.LOCAL\\Administrator"
	role_id   = "${data.vsphere_role.default.id}"
}
`,
		"hashi-dc",
	)
}
