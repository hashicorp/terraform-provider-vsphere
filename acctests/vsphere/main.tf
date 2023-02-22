# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "VSPHERE_LICENSE" {
  sensitive = true
}

variable "VSPHERE_DATACENTER" {}

variable "VSPHERE_CLUSTER" {}

variable "VSPHERE_ESXI_TRUNK_NIC" {}

variable "VSPHERE_RESOURCE_POOL" {}

variable "VSPHERE_DVS_NAME" {}

variable "VSPHERE_NFS_DS_NAME" {}

variable "VSPHERE_PG_NAME" {}

variable "VSPHERE_TEMPLATE" {}

variable "VSPHERE_NAS_HOST" {}

variable "VSPHERE_ESXI1" {}

variable "VSPHERE_ESXI1_PW" {
  sensitive = true
}

variable "VSPHERE_ESXI2" {}

variable "VSPHERE_ESXI2_PW" {
  sensitive = true
}

variable "VSPHERE_ESXI3" {}

variable "VSPHERE_ESXI3_PW" {
  sensitive = true
}

variable "VSPHERE_ESXI4" {}

variable "VSPHERE_ESXI4_PW" {
  sensitive = true
}

data "vsphere_network" "vmnet" {
  datacenter_id = vsphere_datacenter.dc.moid
  name          = "VM Network"
  depends_on = [
    vsphere_distributed_port_group.pg,
    vsphere_distributed_virtual_switch.dvs
  ]
}

resource "vsphere_resource_pool" "pool" {
  name                    = var.VSPHERE_RESOURCE_POOL
  parent_resource_pool_id = vsphere_compute_cluster.compute_cluster.resource_pool_id
}

resource "vsphere_nas_datastore" "ds" {
  name            = var.VSPHERE_NFS_DS_NAME
  host_system_ids = [vsphere_host.host1.id, vsphere_host.host2.id, vsphere_host.host3.id, vsphere_host.host4.id]
  type            = "NFS"
  remote_hosts    = [var.VSPHERE_NAS_HOST]
  remote_path     = "/nfs"
}

resource "vsphere_distributed_virtual_switch" "dvs" {
  name          = var.VSPHERE_DVS_NAME
  datacenter_id = vsphere_datacenter.dc.moid

  uplinks         = ["uplink1", "uplink2"]
  active_uplinks  = ["uplink1"]
  standby_uplinks = ["uplink2"]

  host {
    host_system_id = vsphere_host.host1.id
    devices        = [var.VSPHERE_ESXI_TRUNK_NIC]
  }

  host {
    host_system_id = vsphere_host.host2.id
    devices        = [var.VSPHERE_ESXI_TRUNK_NIC]
  }

  host {
    host_system_id = vsphere_host.host3.id
    devices        = [var.VSPHERE_ESXI_TRUNK_NIC]
  }

  host {
    host_system_id = vsphere_host.host4.id
    devices        = [var.VSPHERE_ESXI_TRUNK_NIC]
  }

  version = "7.0.0"
}

resource "vsphere_distributed_port_group" "pg" {
  name                            = var.VSPHERE_PG_NAME
  distributed_virtual_switch_uuid = vsphere_distributed_virtual_switch.dvs.id

  //vlan_id = var.VSPHERE_PXE_VLAN
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

data "vsphere_host_thumbprint" "esxi2" {
  address  = var.VSPHERE_ESXI2
  insecure = true
}

data "vsphere_host_thumbprint" "esxi3" {
  address  = var.VSPHERE_ESXI3
  insecure = true
}

data "vsphere_host_thumbprint" "esxi4" {
  address  = var.VSPHERE_ESXI4
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

resource "vsphere_host" "host2" {
  hostname   = var.VSPHERE_ESXI2
  username   = "root"
  password   = var.VSPHERE_ESXI2_PW
  license    = vsphere_license.license.license_key
  force      = true
  cluster    = vsphere_compute_cluster.compute_cluster.id
  thumbprint = data.vsphere_host_thumbprint.esxi2.id
}

# Manually enable vsan on this host in the vsphere UI
# Enable vsan in VMKernal Adapters on the management network
resource "vsphere_host" "host3" {
  hostname   = var.VSPHERE_ESXI3
  username   = "root"
  password   = var.VSPHERE_ESXI3_PW
  license    = vsphere_license.license.license_key
  force      = true
  datacenter = vsphere_datacenter.dc.moid
  thumbprint = data.vsphere_host_thumbprint.esxi3.id
}

# Manually enable vSAN on this host in the vSphere UI
# Enable vSAN traffic type in VMKernal adapter on the management network
resource "vsphere_host" "host4" {
  hostname   = var.VSPHERE_ESXI4
  username   = "root"
  password   = var.VSPHERE_ESXI4_PW
  license    = vsphere_license.license.license_key
  force      = true
  datacenter = vsphere_datacenter.dc.moid
  thumbprint = data.vsphere_host_thumbprint.esxi4.id
}

resource "vsphere_virtual_machine" "template" {
  name             = var.VSPHERE_TEMPLATE
  resource_pool_id = vsphere_compute_cluster.compute_cluster.resource_pool_id
  datastore_id     = vsphere_nas_datastore.ds.id
  datacenter_id    = vsphere_datacenter.dc.moid
  host_system_id   = vsphere_host.host2.id

  wait_for_guest_net_timeout = -1

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinuxGuest"

  network_interface {
    network_id = vsphere_distributed_port_group.pg.id
  }

  ovf_deploy {
    remote_ovf_url = "https://storage.googleapis.com/vsphere-acctest/TinyVM/TinyVM.ovf"
    ovf_network_map = {
      "terraform-test-pg" = vsphere_distributed_port_group.pg.id
    }
  }
}

resource "vsphere_virtual_machine" "pxe" {
  name             = "pxe-server"
  resource_pool_id = vsphere_compute_cluster.compute_cluster.resource_pool_id
  datastore_id     = vsphere_nas_datastore.ds.id

  wait_for_guest_net_timeout = -1

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinuxGuest"

  network_interface {
    network_id = data.vsphere_network.vmnet.id
  }
  network_interface {
    network_id = vsphere_distributed_port_group.pg.id
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

data "vsphere_vmfs_disks" "available" {
  host_system_id = vsphere_host.host1.id
  rescan         = true
  filter         = "naa."
}

resource "local_file" "devrc" {
  content = templatefile("./devrc.tpl", {
    vmfs_disk_0 = data.vsphere_vmfs_disks.available.disks[0]
    vmfs_disk_1 = data.vsphere_vmfs_disks.available.disks[1]
  })
  filename = "./devrc"
}

resource "vsphere_content_library" "nested-library" {
  name            = "nested esxi content library"
  storage_backing = [vsphere_nas_datastore.ds.id]
  description     = "https://www.virtuallyghetto.com/2015/04/subscribe-to-vghetto-nested-esxi-template-content-library-in-vsphere-6-0.html"
}

resource "vsphere_content_library_item" "n-esxi" {
  name        = "nested esxi vm"
  description = "nested esxi template"
  library_id  = vsphere_content_library.nested-library.id
  file_url    = "https://s3-us-west-1.amazonaws.com/vghetto-content-library/Nested-ESXi-VM-Template/Nested-ESXi-VM-Template.ovf"
}
