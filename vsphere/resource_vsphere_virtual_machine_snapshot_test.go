package vsphere

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/virtualmachine"
)

func TestAccResourceVSphereVirtualMachineSnapshot_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachineSnapshotPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccResourceVSphereVirtualMachineSnapshotConfig(true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVirtualMachineSnapshotExists("vsphere_virtual_machine_snapshot.snapshot"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine_snapshot.snapshot", "snapshot_name", "terraform-test-snapshot"),
				),
			},
			resource.TestStep{
				Config: testAccResourceVSphereVirtualMachineSnapshotConfig(false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVirtualMachineHasNoSnapshots("vsphere_virtual_machine.vm"),
				),
			},
		},
	})
}

func testAccResourceVSphereVirtualMachineSnapshotPreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_DATACENTER") == "" {
		t.Skip("set VSPHERE_DATACENTER to run vsphere_virtual_machine_snapshot acceptance tests")
	}
	if os.Getenv("VSPHERE_CLUSTER") == "" {
		t.Skip("set VSPHERE_CLUSTER to run vsphere_virtual_machine_snapshot acceptance tests")
	}
	if os.Getenv("VSPHERE_RESOURCE_POOL") == "" {
		t.Skip("set VSPHERE_RESOURCE_POOL to run vsphere_virtual_machine_snapshot acceptance tests")
	}
	if os.Getenv("VSPHERE_NETWORK_LABEL") == "" {
		t.Skip("set VSPHERE_NETWORK_LABEL to run vsphere_virtual_machine_snapshot acceptance tests")
	}
	if os.Getenv("VSPHERE_IPV4_ADDRESS") == "" {
		t.Skip("set VSPHERE_IPV4_ADDRESS to run vsphere_virtual_machine_snapshot acceptance tests")
	}
	if os.Getenv("VSPHERE_IPV4_PREFIX") == "" {
		t.Skip("set VSPHERE_IPV4_PREFIX to run vsphere_virtual_machine_snapshot acceptance tests")
	}
	if os.Getenv("VSPHERE_IPV4_GATEWAY") == "" {
		t.Skip("set VSPHERE_IPV4_GATEWAY to run vsphere_virtual_machine_snapshot acceptance tests")
	}
	if os.Getenv("VSPHERE_DATASTORE") == "" {
		t.Skip("set VSPHERE_DATASTORE to run vsphere_virtual_machine_snapshot acceptance tests")
	}
	if os.Getenv("VSPHERE_TEMPLATE") == "" {
		t.Skip("set VSPHERE_TEMPLATE to run vsphere_virtual_machine_snapshot acceptance tests")
	}
}

func testAccCheckVirtualMachineSnapshotExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Vm Snapshot ID is set")
		}
		client := testAccProvider.Meta().(*VSphereClient).vimClient

		vm, err := virtualmachine.FromUUID(client, rs.Primary.Attributes["virtual_machine_uuid"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout) // This is 5 mins
		defer cancel()
		snapshot, err := vm.FindSnapshot(ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error while getting the snapshot %v", snapshot)
		}

		return nil
	}
}

func testAccCheckVirtualMachineHasNoSnapshots(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VM ID is set")
		}
		client := testAccProvider.Meta().(*VSphereClient).vimClient

		vm, err := virtualmachine.FromUUID(client, rs.Primary.Attributes["uuid"])
		if err != nil {
			return fmt.Errorf("error %s", err)
		}
		props, err := virtualmachine.Properties(vm)
		if err != nil {
			return fmt.Errorf("cannot get properties for virtual machine: %s", err)
		}
		if props.Snapshot != nil {
			return fmt.Errorf("expected VM to not have snapshots, got %#v", props.Snapshot)
		}

		return nil
	}
}

func testAccResourceVSphereVirtualMachineSnapshotConfig(enabled bool) string {
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

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "snapshot_enabled" {
  default = "%t"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
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

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 1024
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label = "disk0"
    size  = "${data.vsphere_virtual_machine.template.disks.0.size}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = true

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway = "${var.ipv4_gateway}"
    }
  }
}

resource "vsphere_virtual_machine_snapshot" "snapshot" {
  count                = "${var.snapshot_enabled == "true" ? 1 : 0 }"
  virtual_machine_uuid = "${vsphere_virtual_machine.vm.uuid}"
  snapshot_name        = "terraform-test-snapshot"
  description          = "Managed by Terraform"
  memory               = true
  quiesce              = true
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		enabled,
	)
}
