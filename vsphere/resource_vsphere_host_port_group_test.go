package vsphere

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/types"
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
		{
			"more complex configuration and overridden attributes",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereHostPortGroupPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereHostPortGroupExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereHostPortGroupConfigWithOverrides(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereHostPortGroupExists(true),
							testAccResourceVSphereHostPortGroupCheckVlan(1000),
							testAccResourceVSphereHostPortGroupCheckEffectiveActive([]string{os.Getenv("VSPHERE_HOST_NIC0")}),
							testAccResourceVSphereHostPortGroupCheckEffectiveStandby([]string{os.Getenv("VSPHERE_HOST_NIC1")}),
							testAccResourceVSphereHostPortGroupCheckEffectivePromisc(true),
						),
					},
				},
			},
		},
		{
			"basic, then complex config",
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
					{
						Config: testAccResourceVSphereHostPortGroupConfigWithOverrides(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereHostPortGroupExists(true),
							testAccResourceVSphereHostPortGroupCheckVlan(1000),
							testAccResourceVSphereHostPortGroupCheckEffectiveActive([]string{os.Getenv("VSPHERE_HOST_NIC0")}),
							testAccResourceVSphereHostPortGroupCheckEffectiveStandby([]string{os.Getenv("VSPHERE_HOST_NIC1")}),
							testAccResourceVSphereHostPortGroupCheckEffectivePromisc(true),
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

// testGetPortGroup is a convenience method to fetch a static port group
// resource for testing.
func testGetPortGroup(s *terraform.State, name, host, datacenter string) (*types.HostPortGroup, error) {
	rs, ok := s.RootModule().Resources[fmt.Sprintf("vsphere_host_port_group.%s", name)]
	if !ok {
		return nil, fmt.Errorf("vsphere_host_port_group.%s not found in state", name)
	}

	client := testAccProvider.Meta().(*govmomi.Client)
	id := rs.Primary.ID
	timeout := time.Minute * 5
	return hostPortGroupFromName(client, id, host, datacenter, timeout)
}

func testAccResourceVSphereHostPortGroupExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		name := "PGTerraformTest"
		id := "pg"
		host := os.Getenv("VSPHERE_ESXI_HOST")
		datacenter := os.Getenv("VSPHERE_DATACENTER")
		_, err := testGetPortGroup(s, id, host, datacenter)
		if err != nil {
			if err.Error() == fmt.Sprintf("port group %s not found on host %s", name, host) && !expected {
				// Expected missing
				return nil
			}
			return err
		}
		if !expected {
			return fmt.Errorf("expected port group %s to still be missing", name)
		}
		return nil
	}
}

func testAccResourceVSphereHostPortGroupCheckVlan(expected int32) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id := "pg"
		host := os.Getenv("VSPHERE_ESXI_HOST")
		datacenter := os.Getenv("VSPHERE_DATACENTER")
		pg, err := testGetPortGroup(s, id, host, datacenter)
		if err != nil {
			return err
		}
		actual := pg.Spec.VlanId
		if expected != actual {
			return fmt.Errorf("expected VLAN ID to be %d, got %d", expected, actual)
		}
		return nil
	}
}

func testAccResourceVSphereHostPortGroupCheckEffectiveActive(expected []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id := "pg"
		host := os.Getenv("VSPHERE_ESXI_HOST")
		datacenter := os.Getenv("VSPHERE_DATACENTER")
		pg, err := testGetPortGroup(s, id, host, datacenter)
		if err != nil {
			return err
		}
		actual := pg.ComputedPolicy.NicTeaming.NicOrder.ActiveNic
		if !reflect.DeepEqual(expected, actual) {
			return fmt.Errorf("expected effective active NICs to be %#v, got %#v", expected, actual)
		}
		return nil
	}
}

func testAccResourceVSphereHostPortGroupCheckEffectiveStandby(expected []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		name := "pg"
		host := os.Getenv("VSPHERE_ESXI_HOST")
		datacenter := os.Getenv("VSPHERE_DATACENTER")
		pg, err := testGetPortGroup(s, name, host, datacenter)
		if err != nil {
			return err
		}
		actual := pg.ComputedPolicy.NicTeaming.NicOrder.StandbyNic
		if !reflect.DeepEqual(expected, actual) {
			return fmt.Errorf("expected effective standby NICs to be %#v, got %#v", expected, actual)
		}
		return nil
	}
}

func testAccResourceVSphereHostPortGroupCheckEffectivePromisc(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		name := "pg"
		host := os.Getenv("VSPHERE_ESXI_HOST")
		datacenter := os.Getenv("VSPHERE_DATACENTER")
		pg, err := testGetPortGroup(s, name, host, datacenter)
		if err != nil {
			return err
		}
		actual := *pg.ComputedPolicy.Security.AllowPromiscuous
		if expected != actual {
			return fmt.Errorf("expected effective allow promiscuous to be %t, got %t", expected, actual)
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

func testAccResourceVSphereHostPortGroupConfigWithOverrides() string {
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

  allow_promiscuous = false
}

resource "vsphere_host_port_group" "pg" {
  name                = "PGTerraformTest"
  host                = "${var.esxi_host}"
  datacenter          = "${var.datacenter}"
  virtual_switch_name = "${vsphere_host_virtual_switch.switch.id}"

  vlan_id = 1000

  active_nics  = ["${var.host_nic0}"]
  standby_nics = ["${var.host_nic1}"]

  allow_promiscuous = true
}
`, os.Getenv("VSPHERE_ESXI_HOST"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_HOST_NIC0"), os.Getenv("VSPHERE_HOST_NIC1"))
}
