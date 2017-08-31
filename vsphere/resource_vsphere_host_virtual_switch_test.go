package vsphere

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccResourceVSphereHostVirtualSwitch(t *testing.T) {
	var tp *testing.T
	testAccResourceVSphereHostVirtualSwitchCases := []struct {
		name     string
		testCase resource.TestCase
	}{
		{
			"basic",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereHostVirtualSwitchPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereHostVirtualSwitchExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereHostVirtualSwitchConfig(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereHostVirtualSwitchExists(true),
						),
					},
				},
			},
		},
		{
			"basic, then remove a NIC",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereHostVirtualSwitchPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereHostVirtualSwitchExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereHostVirtualSwitchConfig(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereHostVirtualSwitchExists(true),
						),
					},
					{
						Config: testAccResourceVSphereHostVirtualSwitchConfigSingleNIC(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereHostVirtualSwitchExists(true),
						),
					},
				},
			},
		},
		{
			"standby with explicit failover order",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereHostVirtualSwitchPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereHostVirtualSwitchExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereHostVirtualSwitchConfigStandbyLink(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereHostVirtualSwitchExists(true),
						),
					},
				},
			},
		},
		{
			"basic, then change to standby with failover order",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereHostVirtualSwitchPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereHostVirtualSwitchExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereHostVirtualSwitchConfig(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereHostVirtualSwitchExists(true),
						),
					},
					{
						Config: testAccResourceVSphereHostVirtualSwitchConfigStandbyLink(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereHostVirtualSwitchExists(true),
						),
					},
				},
			},
		},
	}

	for _, tc := range testAccResourceVSphereHostVirtualSwitchCases {
		t.Run(tc.name, func(t *testing.T) {
			tp = t
			resource.Test(t, tc.testCase)
		})
	}
}

func testAccResourceVSphereHostVirtualSwitchPreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_HOST_NIC0") == "" {
		t.Skip("set VSPHERE_HOST_NIC0 to run vsphere_host_virtual_switch acceptance tests")
	}
	if os.Getenv("VSPHERE_HOST_NIC1") == "" {
		t.Skip("set VSPHERE_HOST_NIC1 to run vsphere_host_virtual_switch acceptance tests")
	}
	if os.Getenv("VSPHERE_ESXI_HOST") == "" {
		t.Skip("set VSPHERE_ESXI_HOST to run vsphere_host_virtual_switch acceptance tests")
	}
}

func testAccResourceVSphereHostVirtualSwitchExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vars, err := testClientVariablesForResource(s, "vsphere_host_virtual_switch.switch")
		if err != nil {
			return errors.New("vsphere_host_virtual_switch.switch not found in state")
		}

		hsID, name, err := splitHostVirtualSwitchID(vars.resourceID)
		if err != nil {
			return err
		}
		ns, err := hostNetworkSystemFromHostSystemID(vars.client, hsID)
		if err != nil {
			return fmt.Errorf("error loading host network system: %s", err)
		}

		_, err = hostVSwitchFromName(vars.client, ns, name)
		if err != nil {
			if err.Error() == fmt.Sprintf("could not find virtual switch %s", name) && !expected {
				// Expected missing
				return nil
			}
			return err
		}
		if !expected {
			return fmt.Errorf("expected vSwitch %s to be missing", name)
		}
		return nil
	}
}

func testAccResourceVSphereHostVirtualSwitchConfig() string {
	return fmt.Sprintf(`
variable "host_nic0" {
  type    = "string"
  default = "%s"
}

variable "host_nic1" {
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

resource "vsphere_host_virtual_switch" "switch" {
  name           = "vSwitchTerraformTest"
  host_system_id = "${data.vsphere_host.esxi_host.id}"

  network_adapters = ["${var.host_nic0}", "${var.host_nic1}"]

  active_nics  = ["${var.host_nic0}", "${var.host_nic1}"]
  standby_nics = []
}
`, os.Getenv("VSPHERE_HOST_NIC0"), os.Getenv("VSPHERE_HOST_NIC1"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
}

func testAccResourceVSphereHostVirtualSwitchConfigSingleNIC() string {
	return fmt.Sprintf(`
variable "host_nic0" {
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

resource "vsphere_host_virtual_switch" "switch" {
  name           = "vSwitchTerraformTest"
  host_system_id = "${data.vsphere_host.esxi_host.id}"

  network_adapters = ["${var.host_nic0}"]

  active_nics  = ["${var.host_nic0}"]
  standby_nics = []
}
`, os.Getenv("VSPHERE_HOST_NIC0"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
}

func testAccResourceVSphereHostVirtualSwitchConfigStandbyLink() string {
	return fmt.Sprintf(`
variable "host_nic0" {
  type    = "string"
  default = "%s"
}

variable "host_nic1" {
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

resource "vsphere_host_virtual_switch" "switch" {
  name           = "vSwitchTerraformTest"
  host_system_id = "${data.vsphere_host.esxi_host.id}"

  network_adapters = ["${var.host_nic0}", "${var.host_nic1}"]

  active_nics    = ["${var.host_nic0}"]
  standby_nics   = ["${var.host_nic1}"]
  teaming_policy = "failover_explicit"
}
`, os.Getenv("VSPHERE_HOST_NIC0"), os.Getenv("VSPHERE_HOST_NIC1"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
}
