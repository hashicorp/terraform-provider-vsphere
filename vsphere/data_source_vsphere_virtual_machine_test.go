// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/testhelper"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "firmware"),
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
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.template", "firmware"),
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
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "firmware"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.uuid", "uuid"),
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
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "firmware"),
					resource.TestCheckResourceAttrSet("data.vsphere_virtual_machine.moid", "moid"),
				),
			},
		},
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
