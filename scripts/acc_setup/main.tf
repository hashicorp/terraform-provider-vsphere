provider "vsphere" {
  user                 = var.VSPHERE_USER
  password             = var.VSPHERE_PASSWORD
  vsphere_server       = var.VSPHERE_SERVER
  allow_unverified_ssl = true
}

variable "VSPHERE_SERVER" {
}

variable "VSPHERE_USER" {
}

variable "VSPHERE_PASSWORD" {
}

variable "VSPHERE_ESXI_USERNAME" {
}

variable "VSPHERE_ESXI1_PASSWORD" {
}

variable "VSPHERE_ESXI2_PASSWORD" {
}

variable "VSPHERE_LICENSE" {
}

variable "VSPHERE_NAS_HOST" {
}

variable "VSPHERE_NFS_PATH" {
}

variable "VSPHERE_ESXI1" {
}

variable "VSPHERE_ESXI2" {
}

variable "VSPHERE_ESXI1_THUMBPRINT" {
}

variable "VSPHERE_ESXI2_THUMBPRINT" {
}

variable "VSPHERE_ESXI_TRUNK_NIC" {
}

variable "VSPHERE_PXE_VLAN" {
}

variable "VSPHERE_VCENTER_ADDRESS" {
}

variable "VSPHERE_VCENTER_NETWORK_PREFIX" {
}

variable "VSPHERE_VCENTER_GATEWAY" {
}

variable "VSPHERE_DATACENTER" {
}

variable "VSPHERE_CLUSTER" {
}

variable "VSPHERE_RESOURCE_POOL" {
}

variable "VSPHERE_DVS_NAME" {
}

variable "VSPHERE_NFS_DS_NAME" {
}

variable "VSPHERE_PG_NAME" {
}

variable "VSPHERE_TEMPLATE" {}

data "vsphere_network" "vmnet" {
  datacenter_id = vsphere_datacenter.dc.moid
  name          = "VM Network"
}

resource "vsphere_datacenter" "dc" {
  name = var.VSPHERE_DATACENTER
}
resource "vsphere_license" "license" {
  license_key = var.VSPHERE_LICENSE
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name          = var.VSPHERE_CLUSTER
  datacenter_id = vsphere_datacenter.dc.moid
  host_managed  = true

  drs_enabled          = true
  drs_automation_level = "fullyAutomated"

  ha_enabled = true
}

resource "vsphere_host" "host1" {
  hostname   = var.VSPHERE_ESXI1
  username   = var.VSPHERE_ESXI_USERNAME
  password   = var.VSPHERE_ESXI1_PASSWORD
  license    = vsphere_license.license.license_key
  force      = true
  cluster    = vsphere_compute_cluster.compute_cluster.id
  thumbprint = var.VSPHERE_ESXI1_THUMBPRINT
}

resource "vsphere_host" "host2" {
  hostname   = var.VSPHERE_ESXI2
  username   = var.VSPHERE_ESXI_USERNAME
  password   = var.VSPHERE_ESXI2_PASSWORD
  license    = vsphere_license.license.license_key
  force      = true
  cluster    = vsphere_compute_cluster.compute_cluster.id
  thumbprint = var.VSPHERE_ESXI2_THUMBPRINT
}

resource "vsphere_resource_pool" "pool" {
  name                    = var.VSPHERE_RESOURCE_POOL
  parent_resource_pool_id = vsphere_compute_cluster.compute_cluster.resource_pool_id
}

resource "vsphere_nas_datastore" "ds" {
  name            = var.VSPHERE_NFS_DS_NAME
  host_system_ids = [vsphere_host.host1.id, vsphere_host.host2.id]
  type            = "NFS"
  remote_hosts    = [var.VSPHERE_NAS_HOST]
  remote_path     = var.VSPHERE_NFS_PATH
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
}

resource "vsphere_distributed_port_group" "pg" {
  name                            = var.VSPHERE_PG_NAME
  distributed_virtual_switch_uuid = vsphere_distributed_virtual_switch.dvs.id

  vlan_id = var.VSPHERE_PXE_VLAN
}

resource "vsphere_virtual_machine" "template" {
  name             = var.VSPHERE_TEMPLATE
  resource_pool_id = vsphere_compute_cluster.compute_cluster.resource_pool_id
  datastore_id     = vsphere_nas_datastore.ds.id
  datacenter_id    = vsphere_datacenter.dc.moid
  host_system_id   = vsphere_host.host1.id

  template                   = true
  wait_for_guest_net_timeout = -1

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = vsphere_distributed_port_group.pg.id
  }

  ovf_deploy {
    remote_ovf_url = "https://acctest-images.storage.googleapis.com/template_test.ovf"
    ovf_network_map = {
      "terraform-test-pg" = vsphere_distributed_port_group.pg.id
    }
  }
  disk {
    label = "disk0"
    size  = 20
  }

  cdrom {
    client_device = true
  }
}

resource "vsphere_virtual_machine_snapshot" "template_snap" {
  virtual_machine_uuid = vsphere_virtual_machine.template.id
  snapshot_name        = "Snapshot1"
  description          = "Snapshot for templates"
  memory               = "true"
  quiesce              = "true"
  remove_children      = "false"
  consolidate          = "true"
}

resource "vsphere_virtual_machine" "pxe" {
  name             = "pxe-server"
  resource_pool_id = vsphere_compute_cluster.compute_cluster.resource_pool_id
  datastore_id     = vsphere_nas_datastore.ds.id

  num_cpus = 2
  memory   = 2048
  guest_id = "ubuntu64Guest"

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

