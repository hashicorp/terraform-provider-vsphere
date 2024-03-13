# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "VSPHERE_LICENSE" {
  sensitive = true
}

variable "VSPHERE_DATACENTER" {
  default = "hashidc"
}

variable "VSPHERE_CLUSTER" {
  default = "c1"
}

variable "VSPHERE_ESXI1" {}

variable "VSPHERE_ESXI1_PW" {
  sensitive = true
}

variable "VSPHERE_ESXI_TRUNK_NIC" {
  default = "vmnic1"
}

variable "VSPHERE_PG_NAME" {
  default = "vmnet"
}

variable "VSPHERE_ESXI1_BOOT_DISK1" {}

resource "vsphere_datacenter" "dc" {
  name = var.VSPHERE_DATACENTER
}

resource "vsphere_license" "license" {
  license_key = var.VSPHERE_LICENSE
}

data "vsphere_host_thumbprint" "esxi1" {
  address  = var.VSPHERE_ESXI1
  insecure = true
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name          = var.VSPHERE_CLUSTER
  datacenter_id = vsphere_datacenter.dc.moid
  host_managed  = true

  drs_enabled          = true
  drs_automation_level = "manual"
  ha_enabled           = false
}

resource "vsphere_host" "host1" {
  hostname   = var.VSPHERE_ESXI1
  username   = "root"
  password   = var.VSPHERE_ESXI1_PW
  license    = vsphere_license.license.license_key
  force      = true
  cluster    = vsphere_compute_cluster.compute_cluster.id
  thumbprint = data.vsphere_host_thumbprint.esxi1.id
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

data "vsphere_vmfs_disks" "boot" {
  host_system_id = vsphere_host.host1.id
  rescan         = true
  filter         = var.VSPHERE_ESXI1_BOOT_DISK1
}

resource "vsphere_vmfs_datastore" "datastore2" {
  name           = "datastore2"
  host_system_id = vsphere_host.host1.id
  disks          = data.vsphere_vmfs_disks.boot.disks
}
