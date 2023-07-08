# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "VSPHERE_ESXI1_BOOT_DISK1_SIZE" {
  default = 100
}

variable "VSPHERE_NFS_DS_NAME" {
  default = "nfs"
}

variable "VSPHERE_PUBLIC_NETWORK" {}

locals {
  vsphere_nas_host = cidrhost(var.VSPHERE_PUBLIC_NETWORK, 4)
}

resource "vsphere_virtual_machine" "nfs" {
  name                 = var.VSPHERE_NFS_DS_NAME
  datacenter_id        = data.vsphere_datacenter.dc.id
  datastore_id         = data.vsphere_ovf_vm_template.ubuntu.datastore_id
  host_system_id       = data.vsphere_ovf_vm_template.ubuntu.host_system_id
  resource_pool_id     = data.vsphere_ovf_vm_template.ubuntu.resource_pool_id
  num_cpus             = data.vsphere_ovf_vm_template.ubuntu.num_cpus
  num_cores_per_socket = data.vsphere_ovf_vm_template.ubuntu.num_cores_per_socket
  memory               = data.vsphere_ovf_vm_template.ubuntu.memory
  guest_id             = data.vsphere_ovf_vm_template.ubuntu.guest_id
  firmware             = data.vsphere_ovf_vm_template.ubuntu.firmware
  scsi_type            = data.vsphere_ovf_vm_template.ubuntu.scsi_type
  nested_hv_enabled    = data.vsphere_ovf_vm_template.ubuntu.nested_hv_enabled

  disk {
    label = "disk0"
    size  = var.VSPHERE_ESXI1_BOOT_DISK1_SIZE
  }

  dynamic "network_interface" {
    for_each = data.vsphere_ovf_vm_template.ubuntu.ovf_network_map
    content {
      network_id = network_interface.value
    }
  }

  wait_for_guest_net_timeout = 0
  wait_for_guest_ip_timeout  = 0

  ovf_deploy {
    allow_unverified_ssl_cert = false
    remote_ovf_url            = data.vsphere_ovf_vm_template.ubuntu.remote_ovf_url
    disk_provisioning         = data.vsphere_ovf_vm_template.ubuntu.disk_provisioning
    ovf_network_map           = data.vsphere_ovf_vm_template.ubuntu.ovf_network_map
  }

  cdrom {
    client_device = true
  }

  vapp {
    properties = {
      "instance-id" = var.VSPHERE_NFS_DS_NAME
      "hostname"    = var.VSPHERE_NFS_DS_NAME
      "password"    = "ubuntu"
      "user-data" = base64encode(
        templatefile("${path.module}/nfs-cloud-config.yaml.tpl",
          {
            ip      = local.vsphere_nas_host
            gateway = cidrhost(var.VSPHERE_PUBLIC_NETWORK, 1)
          }
        )
      )
    }
  }
}

resource "time_sleep" "wait_180_seconds_nfs" {
  depends_on = [vsphere_virtual_machine.nfs]

  create_duration = "180s"
}

resource "vsphere_nas_datastore" "ds" {
  depends_on = [time_sleep.wait_180_seconds_nfs]

  name            = var.VSPHERE_NFS_DS_NAME
  host_system_ids = [data.vsphere_host.host1.id] // TODO: needs to be networked privately for nested ESXIs to connect to it
  type            = "NFS"
  remote_hosts    = [local.vsphere_nas_host]
  remote_path     = "/nfs"
}
