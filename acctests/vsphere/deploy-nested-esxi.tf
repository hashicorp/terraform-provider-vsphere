# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "NESTED_COUNT" {
  default = 3
}

variable "VSPHERE_PRIVATE_NETWORK" {}

variable "PRIV_KEY" {}

resource "vsphere_virtual_machine" "nested-esxi" {
  count = var.NESTED_COUNT

  name                 = "${data.vsphere_ovf_vm_template.nested-esxi.name}-${count.index + 1}"
  datacenter_id        = vsphere_datacenter.dc.moid
  datastore_id         = data.vsphere_ovf_vm_template.nested-esxi.datastore_id
  host_system_id       = data.vsphere_ovf_vm_template.nested-esxi.host_system_id
  resource_pool_id     = data.vsphere_ovf_vm_template.nested-esxi.resource_pool_id
  num_cpus             = data.vsphere_ovf_vm_template.nested-esxi.num_cpus
  num_cores_per_socket = data.vsphere_ovf_vm_template.nested-esxi.num_cores_per_socket
  memory               = data.vsphere_ovf_vm_template.nested-esxi.memory
  guest_id             = data.vsphere_ovf_vm_template.nested-esxi.guest_id
  firmware             = data.vsphere_ovf_vm_template.nested-esxi.firmware
  scsi_type            = data.vsphere_ovf_vm_template.nested-esxi.scsi_type
  nested_hv_enabled    = data.vsphere_ovf_vm_template.nested-esxi.nested_hv_enabled
  dynamic "network_interface" {
    for_each = data.vsphere_ovf_vm_template.nested-esxi.ovf_network_map
    content {
      network_id = network_interface.value
    }
  }
  wait_for_guest_net_timeout = 0
  wait_for_guest_ip_timeout  = 0

  ovf_deploy {
    allow_unverified_ssl_cert = false
    remote_ovf_url            = data.vsphere_ovf_vm_template.nested-esxi.remote_ovf_url
    ip_protocol               = "IPV4"
    ip_allocation_policy      = "STATIC_MANUAL"
    disk_provisioning         = data.vsphere_ovf_vm_template.nested-esxi.disk_provisioning
    ovf_network_map           = data.vsphere_ovf_vm_template.nested-esxi.ovf_network_map
  }

  vapp {
    properties = {
      "guestinfo.hostname"   = "nested-${count.index + 1}.test.local",
      "guestinfo.ipaddress"  = cidrhost(var.VSPHERE_PRIVATE_NETWORK, count.index + 3),
      "guestinfo.netmask"    = "255.255.255.248",
      "guestinfo.gateway"    = cidrhost(var.VSPHERE_PRIVATE_NETWORK, 1),
      "guestinfo.dns"        = cidrhost(var.VSPHERE_PRIVATE_NETWORK, 1),
      "guestinfo.domain"     = "test.local",
      "guestinfo.ntp"        = cidrhost(var.VSPHERE_PRIVATE_NETWORK, 1),
      "guestinfo.ssh"        = "True",
      "guestinfo.password"   = "VMware1!",
      "guestinfo.createvmfs" = "False",
    }
  }

  lifecycle {
    ignore_changes = [
      annotation,
      disk[0].io_share_count,
      disk[1].io_share_count,
      disk[2].io_share_count,
      vapp[0].properties,
    ]
  }
}

// TODO: figure out why builtin network waiter not working
resource "time_sleep" "wait_180_seconds" {
  depends_on = [vsphere_virtual_machine.nested-esxi]
  triggers = {
    change_in_hostcount = length(vsphere_virtual_machine.nested-esxi)
  }
  create_duration = "180s"
}

locals {
  hosts = join(" ", [for h in vsphere_virtual_machine.nested-esxi : h.vapp[0].properties["guestinfo.ipaddress"]])
}

resource "null_resource" "retrieve_thumbprints" {
  depends_on = [time_sleep.wait_180_seconds]

  triggers = {
    hosts = local.hosts
  }

  # this script will ssh into the physical ESXI, and then use openssl to retrieve the thumbprints via the private IPs
  provisioner "local-exec" {
    command = <<-EOT
      ssh -o StrictHostKeyChecking=no -i "${var.PRIV_KEY}" "root@${var.VSPHERE_ESXI1}" <<-'SSHCOMMANDS' > thumbprints.txt
        #!/bin/sh
        esxcli network firewall ruleset set -e true -r httpClient
        hosts="${local.hosts}"
        for host in $hosts; do
          echo | openssl s_client -connect $host:443 -servername $host -showcerts 2>/dev/null | openssl x509 -noout -fingerprint -sha1 | awk -F'=' '{print $2}'
        done
      SSHCOMMANDS
    EOT
  }
}

# load the thumbprints from the file
data "external" "thumbprints" {
  depends_on = [null_resource.retrieve_thumbprints]

  program = ["sh", "-c", "jq -R -s '{thumbprints: .}' ${path.module}/thumbprints.txt"]
}

locals {
  thumbprints = split("\n", data.external.thumbprints.result["thumbprints"])
}

resource "vsphere_host" "nested-esxi" {
  depends_on = [null_resource.retrieve_thumbprints]

  count = var.NESTED_COUNT

  hostname   = vsphere_virtual_machine.nested-esxi[count.index].vapp[0].properties["guestinfo.ipaddress"]
  username   = "root"
  password   = "VMware1!"
  license    = vsphere_license.license.license_key
  force      = true
  cluster    = vsphere_compute_cluster.compute_cluster.id
  thumbprint = local.thumbprints[count.index]
}
