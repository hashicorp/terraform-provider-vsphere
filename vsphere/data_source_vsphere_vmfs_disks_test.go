package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceVSphereVmfsDisks_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereVmfsDisksPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereVmfsDisksConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckOutput("found", "true"),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereVmfsDisks_withRegularExpression(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereVmfsDisksPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereVmfsDisksConfigRegexp(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckOutput("expected_length", "true"),
				),
			},
		},
	})
}

func testAccDataSourceVSphereVmfsDisksPreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_ESXI_HOST") == "" {
		t.Skip("set VSPHERE_ESXI_HOST to run vsphere_vmfs_disks acceptance tests")
	}
	if os.Getenv("VSPHERE_VMFS_EXPECTED") == "" {
		t.Skip("set VSPHERE_VMFS_EXPECTED to run vsphere_vmfs_disks acceptance tests")
	}
	if os.Getenv("VSPHERE_VMFS_REGEXP") == "" {
		t.Skip("set VSPHERE_VMFS_REGEXP to run vsphere_vmfs_disks acceptance tests")
	}
}

func testAccDataSourceVSphereVmfsDisksConfig() string {
	return fmt.Sprintf(`
data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_host" "esxi_host" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

data "vsphere_vmfs_disks" "available" {
  host_system_id = "${data.vsphere_host.esxi_host.id}"
  rescan         = true
}

output "found" {
  value = "${contains(data.vsphere_vmfs_disks.available.disks, "%s")}"
}
`, os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"), os.Getenv("VSPHERE_VMFS_EXPECTED"))
}

func testAccDataSourceVSphereVmfsDisksConfigRegexp() string {
	return fmt.Sprintf(`
variable "regexp" {
  type    = "string"
  default = "%s"
}

data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_host" "esxi_host" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

data "vsphere_vmfs_disks" "available" {
  host_system_id = "${data.vsphere_host.esxi_host.id}"
  rescan         = true
  filter         = "${var.regexp}"
}

output "expected_length" {
  value = "${length(data.vsphere_vmfs_disks.available.disks) == 3 ? "true" : "false" }"
}
`, os.Getenv("VSPHERE_VMFS_REGEXP"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
}
