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
