# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "VSPHERE_DATACENTER" {}

variable "VSPHERE_ESXI1" {}

variable "VSPHERE_CLUSTER" {}

variable "VSPHERE_PG_NAME" {}

variable "VSPHERE_LICENSE" {
  sensitive = true
}

data "vsphere_license" "license" {
  license_key = var.VSPHERE_LICENSE
}

data "vsphere_datacenter" "dc" {
  name = var.VSPHERE_DATACENTER
}

data "vsphere_host" "host1" {
  name          = var.VSPHERE_ESXI1
  datacenter_id = data.vsphere_datacenter.dc.id
}

data "vsphere_compute_cluster" "compute_cluster" {
  name          = var.VSPHERE_CLUSTER
  datacenter_id = data.vsphere_datacenter.dc.id
}

data "vsphere_network" "public" {
  name          = "VM Network"
  datacenter_id = data.vsphere_datacenter.dc.id
}

data "vsphere_network" "pg" {
  name          = var.VSPHERE_PG_NAME
  datacenter_id = data.vsphere_datacenter.dc.id
}

data "vsphere_datastore" "datastore2" {
  name          = "datastore2"
  datacenter_id = data.vsphere_datacenter.dc.id
}
