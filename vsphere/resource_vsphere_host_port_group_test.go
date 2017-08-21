package vsphere

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
)

func TestAccResourceVSphereHostPortGroup(t *testing.T) {
	var tp *testing.T
	testAccResourceVSphereHostPortGroupCases := []struct {
		name     string
		testCase resource.TestCase
	}{
		{
			"basic, inherited policy",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereHostPortGroupPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereHostPortGroupExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereHostPortGroupConfig(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereHostPortGroupExists(true),
						),
					},
				},
			},
		},
	}

	for _, tc := range testAccResourceVSphereHostPortGroupCases {
		t.Run(tc.name, func(t *testing.T) {
			tp = t
			resource.Test(t, tc.testCase)
		})
	}
}

func testAccResourceVSphereHostPortGroupPreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_HOST_NIC0") == "" {
		t.Skip("set VSPHERE_HOST_NIC0 to run vsphere_host_port_group acceptance tests")
	}
	if os.Getenv("VSPHERE_HOST_NIC1") == "" {
		t.Skip("set VSPHERE_HOST_NIC1 to run vsphere_host_port_group acceptance tests")
	}
	if os.Getenv("VSPHERE_ESXI_HOST") == "" {
		t.Skip("set VSPHERE_ESXI_HOST to run vsphere_host_port_group acceptance tests")
	}
}

func testAccResourceVSphereHostPortGroupExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["vsphere_host_port_group.pg"]
		if !ok {
			return errors.New("vsphere_host_port_group.pg not found in state")
		}

		client := testAccProvider.Meta().(*govmomi.Client)
		id := rs.Primary.ID
		host := os.Getenv("VSPHERE_ESXI_HOST")
		datacenter := os.Getenv("VSPHERE_DATACENTER")
		timeout := time.Minute * 5
		_, err := hostPortGroupFromName(client, id, host, datacenter, timeout)
		if err != nil {
			if err.Error() == fmt.Sprintf("port group %s not found on host %s", id, host) && !expected {
				// Expected missing
				return nil
			}
			return err
		}
		if !expected {
			return fmt.Errorf("expected port group %s to still be missing", id)
		}
		return nil
	}
}

func testAccResourceVSphereHostPortGroupConfig() string {
	return fmt.Sprintf(`
variable "esxi_host" {
  type    = "string"
  default = "%s"
}

variable "datacenter" {
  type    = "string"
  default = "%s"
}

variable "host_nic0" {
  type    = "string"
  default = "%s"
}

variable "host_nic1" {
  type    = "string"
  default = "%s"
}

resource "vsphere_host_virtual_switch" "switch" {
  name             = "vSwitchTerraformTest"
  host             = "${var.esxi_host}"
  datacenter       = "${var.datacenter}"
  network_adapters = ["${var.host_nic0}", "${var.host_nic1}"]

  active_nics  = ["${var.host_nic0}", "${var.host_nic1}"]
  standby_nics = []
}

resource "vsphere_host_port_group" "pg" {
  name                = "PGTerraformTest"
  host                = "${var.esxi_host}"
  datacenter          = "${var.datacenter}"
  virtual_switch_name = "${vsphere_host_virtual_switch.switch.id}"
}
`, os.Getenv("VSPHERE_ESXI_HOST"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_HOST_NIC0"), os.Getenv("VSPHERE_HOST_NIC1"))
}
