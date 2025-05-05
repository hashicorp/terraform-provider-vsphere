// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/testhelper"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceVSphereVirtualMachine_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccDataSourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereVirtualMachineConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.vsphere_virtual_machine.template",
						"id",
						regexp.MustCompile("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$")),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "guest_id"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "scsi_type"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "memory"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "memory_reservation_locked_to_max"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "num_cpus"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "num_cores_per_socket"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "firmware"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "hardware_version"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.#"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.0.size"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.0.eagerly_scrub"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.0.thin_provisioned"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.0.unit_number"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.0.label"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interface_types.#"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interfaces.#"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interfaces.0.adapter_type"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interfaces.0.bandwidth_limit"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interfaces.0.bandwidth_reservation"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interfaces.0.bandwidth_share_level"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interfaces.0.bandwidth_share_count"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interfaces.0.mac_address"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interfaces.0.network_id"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "instance_uuid"),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereVirtualMachine_noDatacenterAndAbsolutePath(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccDataSourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereVirtualMachineConfigAbsolutePath(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.vsphere_virtual_machine.template",
						"id",
						regexp.MustCompile("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$")),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "guest_id"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "scsi_type"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "memory"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "num_cpus"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "num_cores_per_socket"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "firmware"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "hardware_version"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.#"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.0.size"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.0.eagerly_scrub"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.0.thin_provisioned"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.0.unit_number"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "disks.0.label"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interface_types.#"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interfaces.#"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interfaces.0.adapter_type"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interfaces.0.bandwidth_limit"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interfaces.0.bandwidth_reservation"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interfaces.0.bandwidth_share_level"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interfaces.0.bandwidth_share_count"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interfaces.0.mac_address"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "network_interfaces.0.network_id"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "instance_uuid"),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereVirtualMachine_uuid(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccDataSourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereVirtualMachineConfigUUID(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.vsphere_virtual_machine.uuid",
						"id",
						regexp.MustCompile("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$")),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "guest_id"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "scsi_type"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "memory"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "num_cpus"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "num_cores_per_socket"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "firmware"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "hardware_version"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "disks.#"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "disks.0.size"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "disks.0.eagerly_scrub"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "disks.0.thin_provisioned"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "disks.0.unit_number"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "disks.0.label"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "network_interface_types.#"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "network_interfaces.#"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "network_interfaces.0.adapter_type"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "network_interfaces.0.bandwidth_limit"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "network_interfaces.0.bandwidth_reservation"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "network_interfaces.0.bandwidth_share_level"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "network_interfaces.0.bandwidth_share_count"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "network_interfaces.0.mac_address"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "network_interfaces.0.network_id"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "uuid"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "instance_uuid"),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereVirtualMachine_moid(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccDataSourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereVirtualMachineConfigMOID(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.vsphere_virtual_machine.moid",
						"id",
						regexp.MustCompile("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$")),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "guest_id"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "scsi_type"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "memory"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "num_cpus"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "num_cores_per_socket"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "firmware"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "hardware_version"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "disks.#"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "disks.0.size"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "disks.0.eagerly_scrub"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "disks.0.thin_provisioned"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "disks.0.unit_number"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "disks.0.label"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "network_interface_types.#"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "network_interfaces.#"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "network_interfaces.0.adapter_type"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "network_interfaces.0.bandwidth_limit"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "network_interfaces.0.bandwidth_reservation"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "network_interfaces.0.bandwidth_share_level"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "network_interfaces.0.bandwidth_share_count"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "network_interfaces.0.mac_address"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "network_interfaces.0.network_id"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "instance_uuid"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "moid"),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereVirtualMachine_nameAndFolder(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccDataSourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccDataSourceVirtualMachineFolder(),
			Check: resource.ComposeTestCheckFunc(
				resource.TestMatchResourceAttr(
					"data.vsphere_virtual_machine.vm1",
					"id",
					regexp.MustCompile("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$")),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "guest_id"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "scsi_type"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "memory"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "num_cpus"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "num_cores_per_socket"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "firmware"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "hardware_version"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "disks.#"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "disks.0.size"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "disks.0.eagerly_scrub"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "disks.0.thin_provisioned"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "disks.0.unit_number"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "disks.0.label"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "network_interface_types.#"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "network_interfaces.#"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "network_interfaces.0.adapter_type"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "network_interfaces.0.bandwidth_limit"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "network_interfaces.0.bandwidth_reservation"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "network_interfaces.0.bandwidth_share_level"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "network_interfaces.0.bandwidth_share_count"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "network_interfaces.0.mac_address"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "network_interfaces.0.network_id"),
				resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.vm1", "instance_uuid")),
		}},
	})
}

func testAccDataSourceVSphereVirtualMachinePreCheck(t *testing.T) {
	if os.Getenv("TF_VAR_VSPHERE_DATACENTER") == "" {
		t.Skip("set TF_VAR_VSPHERE_DATACENTER to run vsphere_virtual_machine data source acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_TEMPLATE") == "" {
		t.Skip("set TF_VAR_VSPHERE_TEMPLATE to run vsphere_virtual_machine data source acceptance tests")
	}
}

func testAccDataSourceVSphereVirtualMachineConfigUUID() string {
	return fmt.Sprintf(`
%s

variable "template" {
  default = "%s"
}

data "vsphere_virtual_machine" "template" {
  name          = var.template
  datacenter_id = data.vsphere_datacenter.rootdc1.id
}

data "vsphere_virtual_machine" "uuid" {
  uuid = data.vsphere_virtual_machine.template.uuid
}
`,
		testhelper.CombineConfigs(testhelper.ConfigDataRootDC1(), testhelper.ConfigDataRootPortGroup1()),
		os.Getenv("TF_VAR_VSPHERE_TEMPLATE"),
	)
}

func testAccDataSourceVSphereVirtualMachineConfigMOID() string {
	return fmt.Sprintf(`
%s

variable "template" {
  default = "%s"
}

data "vsphere_virtual_machine" "template" {
  name          = var.template
  datacenter_id = data.vsphere_datacenter.rootdc1.id
}

data "vsphere_virtual_machine" "moid" {
  moid = data.vsphere_virtual_machine.template.moid
}
`,
		testhelper.CombineConfigs(testhelper.ConfigDataRootDC1(), testhelper.ConfigDataRootPortGroup1()),
		os.Getenv("TF_VAR_VSPHERE_TEMPLATE"),
	)
}

func testAccDataSourceVSphereVirtualMachineConfig() string {
	return fmt.Sprintf(`
%s

variable "template" {
  default = "%s"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.rootdc1.id}"
}
`,
		testhelper.CombineConfigs(testhelper.ConfigDataRootDC1(), testhelper.ConfigDataRootPortGroup1()),
		os.Getenv("TF_VAR_VSPHERE_TEMPLATE"),
	)
}

func testAccDataSourceVSphereVirtualMachineConfigAbsolutePath() string {
	return fmt.Sprintf(`
%s

variable "template" {
  default = "%s"
}

data "vsphere_virtual_machine" "template" {
  name = "/${data.vsphere_datacenter.rootdc1.name}/vm/${var.template}"
}
`,
		testhelper.CombineConfigs(testhelper.ConfigDataRootDC1(), testhelper.ConfigDataRootPortGroup1()),
		os.Getenv("TF_VAR_VSPHERE_TEMPLATE"),
	)
}

func testAccDataSourceVirtualMachineFolder() string {
	return fmt.Sprintf(`
	%s

	resource "vsphere_folder" "new_vm_folder" {
		path		  = "new-vm-folder"
		datacenter_id = data.vsphere_datacenter.rootdc1.id
		type		  = "vm"

	}

	resource "vsphere_virtual_machine" "vm" {
	  name             = "foo"
	  resource_pool_id = data.vsphere_compute_cluster.rootcompute_cluster1.resource_pool_id
	  folder 		   = vsphere_folder.new_vm_folder.path
	  datastore_id     = data.vsphere_datastore.rootds1.id
	  num_cpus         = 1
	  memory           = 1024
	  guest_id         = "otherLinux64Guest"
	  network_interface {
		network_id = data.vsphere_network.network1.id
	  }
	  disk {
		label = "disk0"
		size  = 10
	  }
	 wait_for_guest_ip_timeout = 0
	 wait_for_guest_net_timeout  = 0
	}

	data vsphere_virtual_machine "vm1" {
		name 		  = vsphere_virtual_machine.vm.name
		datacenter_id = data.vsphere_datacenter.rootdc1.id
		folder 		  = vsphere_folder.new_vm_folder.path
	}


`, testhelper.CombineConfigs(
		testhelper.ConfigDataRootDC1(),
		testhelper.ConfigDataRootPortGroup1(),
		testhelper.ConfigDataRootComputeCluster1(),
		testhelper.ConfigDataRootDS1(),
	))

}
