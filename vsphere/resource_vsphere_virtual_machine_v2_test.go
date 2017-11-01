package vsphere

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccResourceVSphereVirtualMachineV2(t *testing.T) {
	var tp *testing.T
	testAccResourceVSphereVirtualMachineV2Cases := []struct {
		name     string
		testCase resource.TestCase
	}{
		{
			"basic",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereVirtualMachineV2CheckExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereVirtualMachineV2ConfigBasic(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineV2CheckExists(true),
						),
					},
				},
			},
		},
	}

	for _, tc := range testAccResourceVSphereVirtualMachineV2Cases {
		t.Run(tc.name, func(t *testing.T) {
			tp = t
			resource.Test(t, tc.testCase)
		})
	}
}

func testAccResourceVSphereVirtualMachineV2CheckExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := testGetVirtualMachineV2(s, "vm")
		if err != nil {
			if ok, _ := regexp.MatchString("virtual machine with UUID \"[-a-f0-9]+\" not found", err.Error()); ok && !expected {
				// Expected missing
				return nil
			}
			return err
		}
		if !expected {
			return errors.New("expected VM to be missing")
		}
		return nil
	}
}

func testAccResourceVSphereVirtualMachineV2ConfigBasic() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine_v2" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 1024
  guest_id = "other3xLinux64Guest"

  network_interface {
    index      = 0
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    index = 0
    size  = 20
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}
