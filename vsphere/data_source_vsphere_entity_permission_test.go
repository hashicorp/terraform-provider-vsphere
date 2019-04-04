package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceVSphereEntityPermission_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereEntityPermissionPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereEntityPermissionConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.vsphere_entity_permission.terraform-test-entity-permission", "role_id", "-1"),
				),
			},
		},
	})
}

func testAccDataSourceVSphereEntityPermissionConfig() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "vsphere_user" {
  default = "%s"
}

variable "vsphere_role" {
	default = "%s"
}

variable "vsphere_datastore" {
	default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_role" "role" {
  name = "${var.vsphere_role}"
}

data "vsphere_datastore" "datastore" {
  name = "${var.vsphere_datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_entity_permission" "terraform-test-entity-permission" {
  principal   = "${var.vsphere_user}"
  entity_id   = "${data.vsphere_datastore.datastore.id}"
	entity_type = "Datastore"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_ENTITY_USER"),
		os.Getenv("VSPHERE_ROLE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccDataSourceVSphereEntityPermissionPreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_DATACENTER") == "" {
		t.Skip("set VSPHERE_DATACENTER to run vsphere_entity_permission acceptance tests")
	}
	if os.Getenv("VSPHERE_ENTITY_USER") == "" {
		t.Skip("set VSPHERE_ENTITY_USER to run vsphere_entity_permission acceptance tests")
	}
	if os.Getenv("VSPHERE_ROLE") == "" {
		t.Skip("set VSPHERE_ROLE to run vsphere_entity_permission acceptance tests")
	}
	if os.Getenv("VSPHERE_DATASTORE") == "" {
		t.Skip("set VSPHERE_DATASTORE to run vsphere_entity_permission acceptance tests")
	}
}
