# © Broadcom. All Rights Reserved.
# The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
# SPDX-License-Identifier: MPL-2.0

variable "VSPHERE_RESOURCE_POOL" {
  default = "hashi-resource-pool"
}

variable "VSPHERE_TEMPLATE" {
  default = "tfvsphere_template"
}

variable "VSPHERE_TEST_OVF" {
  default = "https://storage.googleapis.com/vsphere-acctest/TinyVM/TinyVM.ovf"
}

variable "VSPHERE_VM_V1_PATH" {
  default = "test-vm"
}

variable "VSPHERE_VMFS_REGEXP" {
  default = "naa."
}

data "vsphere_vmfs_disks" "available" {
  host_system_id = data.vsphere_host.host1.id
  rescan         = true
  filter         = var.VSPHERE_VMFS_REGEXP
}

resource "vsphere_resource_pool" "pool" {
  name                    = var.VSPHERE_RESOURCE_POOL
  parent_resource_pool_id = data.vsphere_compute_cluster.compute_cluster.resource_pool_id
}

resource "vsphere_virtual_machine" "template" {
  name             = var.VSPHERE_TEMPLATE
  resource_pool_id = data.vsphere_compute_cluster.compute_cluster.resource_pool_id
  datastore_id     = vsphere_nas_datastore.ds.id
  datacenter_id    = data.vsphere_datacenter.dc.id
  host_system_id   = data.vsphere_host.host1.id

  wait_for_guest_net_timeout = -1

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinuxGuest"

  network_interface {
    network_id = data.vsphere_network.pg.id
  }

  ovf_deploy {
    remote_ovf_url = var.VSPHERE_TEST_OVF
    ovf_network_map = {
      "${var.VSPHERE_PG_NAME}" = data.vsphere_network.pg.id
    }
  }
}

resource "vsphere_virtual_machine" "test-vm" {
  name             = var.VSPHERE_VM_V1_PATH
  resource_pool_id = data.vsphere_compute_cluster.compute_cluster.resource_pool_id
  datastore_id     = vsphere_nas_datastore.ds.id

  wait_for_guest_net_timeout = -1

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinuxGuest"

  network_interface {
    network_id = data.vsphere_network.pg.id
  }
  clone {
    template_uuid = vsphere_virtual_machine.template.id
  }

  disk {
    label = "disk0"
    size  = 20
  }

  cdrom {
    client_device = true
  }
}
