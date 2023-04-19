# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "VSPHERE_ESXI_TRUNK_NIC" {
  default = "vmnic1"
}

variable "VSPHERE_PG_NAME" {
  default = "vmnet"
}

variable "VSPHERE_VMFS_REGEXP" {
  default = "naa."
}

data "vsphere_vmfs_disks" "available" {
  host_system_id = vsphere_host.host1.id
  rescan         = true
  filter         = var.VSPHERE_VMFS_REGEXP
}

# TODO: use datastore1 instead for nested esxi VMs
resource "vsphere_vmfs_datastore" "nested-esxi" {
  name           = "nested-esxi"
  host_system_id = vsphere_host.host1.id
  disks          = data.vsphere_vmfs_disks.available.disks
}

resource "vsphere_host_virtual_switch" "switch" {
  name           = "terraform-test"
  host_system_id = vsphere_host.host1.id

  network_adapters = [var.VSPHERE_ESXI_TRUNK_NIC]

  active_nics = [var.VSPHERE_ESXI_TRUNK_NIC]
}

resource "vsphere_host_port_group" "pg" {
  name                = var.VSPHERE_PG_NAME
  host_system_id      = vsphere_host.host1.id
  virtual_switch_name = vsphere_host_virtual_switch.switch.name

  allow_promiscuous      = true
  allow_mac_changes      = true
  allow_forged_transmits = true
}

// TODO: why can't the id exported from vsphere_host_port_group be used directly?
data "vsphere_network" "pg" {
  depends_on = [vsphere_host_port_group.pg, vsphere_host_virtual_switch.switch]

  name          = vsphere_host_port_group.pg.name
  datacenter_id = vsphere_datacenter.dc.moid
}

data "vsphere_ovf_vm_template" "nested-esxi" {
  name              = "Nested-ESXi-7.0"
  disk_provisioning = "thin"
  resource_pool_id  = vsphere_compute_cluster.compute_cluster.resource_pool_id
  datastore_id      = vsphere_vmfs_datastore.nested-esxi.id
  host_system_id    = vsphere_host.host1.id
  remote_ovf_url    = "https://download3.vmware.com/software/vmw-tools/nested-esxi/Nested_ESXi7.0u3_Appliance_Template_v1.ova"
  ovf_network_map = {
    "${vsphere_host_port_group.pg.name}" = data.vsphere_network.pg.id
    "VM Network"                         = data.vsphere_network.pg.id # second NIC for testing
  }
}
