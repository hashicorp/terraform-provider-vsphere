# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "PACKET_AUTH" {
  sensitive = true
}

variable "PACKET_PROJECT" {}

variable "PRIV_KEY" {}

variable "VCSA_DEPLOY_PATH" {}

variable "ESXI_VERSION" {
  default = "vmware_esxi_7_0"
}

variable "PACKET_FACILITY" {
  default = "sv"
}

variable "ESXI_PLAN" {
  default = "c3.medium.x86"
}

variable "STORAGE_PLAN" {
  default = "c3.small.x86"
}

variable "LAB_PREFIX" {
  default = ""
}

provider "equinix" {
  auth_token = var.PACKET_AUTH
}

resource "equinix_metal_device" "esxi1" {
  hostname         = "${var.LAB_PREFIX}e.test.local"
  plan             = var.ESXI_PLAN
  metro            = var.PACKET_FACILITY
  operating_system = var.ESXI_VERSION
  billing_cycle    = "hourly"
  project_id       = var.PACKET_PROJECT
}

resource "equinix_metal_device_network_type" "esxi1" {
  device_id = equinix_metal_device.esxi1.id
  type      = "hybrid"
}

resource "equinix_metal_vlan" "vmvlan" {
  metro      = var.PACKET_FACILITY
  project_id = var.PACKET_PROJECT
}

resource "time_sleep" "wait_120_seconds" {
  depends_on = [equinix_metal_device_network_type.esxi1]

  create_duration = "120s"
}

resource "equinix_metal_port_vlan_attachment" "esxi1" {
  depends_on = [time_sleep.wait_120_seconds]

  device_id = equinix_metal_device.esxi1.id
  port_name = "eth1"
  vlan_vnid = equinix_metal_vlan.vmvlan.vxlan
}

resource "equinix_metal_device" "storage1" {
  hostname         = "${var.LAB_PREFIX}storage1.test.local"
  plan             = var.STORAGE_PLAN
  metro            = var.PACKET_FACILITY
  operating_system = "ubuntu_20_04"
  billing_cycle    = "hourly"
  project_id       = var.PACKET_PROJECT
  provisioner "remote-exec" {
    inline = [
      "mkdir -p /nfs/ds1 /nfs/ds2 /nfs/ds3",
      "apt-get update",
      "apt-get install nfs-common nfs-kernel-server -y",
      "echo \"/nfs *(rw,no_root_squash)\" > /etc/exports",
      "echo \"/nfs/ds1 *(rw,no_root_squash)\" >> /etc/exports",
      "echo \"/nfs/ds2 *(rw,no_root_squash)\" >> /etc/exports",
      "echo \"/nfs/ds3 *(rw,no_root_squash)\" >> /etc/exports",
      "exportfs -a",
    ]
    connection {
      host        = equinix_metal_device.storage1.network.0.address
      private_key = file(var.PRIV_KEY)
    }
  }
}

resource "local_file" "vcsa_template" {
  content = templatefile("${path.cwd}/vcsa_deploy.json", {
    hostname       = equinix_metal_device.esxi1.network.0.address
    password       = equinix_metal_device.esxi1.root_password
    ip_address     = cidrhost("${equinix_metal_device.esxi1.network.0.address}/${equinix_metal_device.esxi1.network.0.cidr}", 3)
    ip_prefix      = equinix_metal_device.esxi1.network.0.cidr
    gateway        = cidrhost("${equinix_metal_device.esxi1.network.0.address}/${equinix_metal_device.esxi1.network.0.cidr}", 1)
    vcenter_fqdn   = "vcenter.test.local"
    admin_password = "Password123!"
  })
  filename = "./tmp/vcsa.json"
  provisioner "local-exec" {
    command = "sleep 300; TERM=xterm-256color ${var.VCSA_DEPLOY_PATH} install --accept-eula --acknowledge-ceip --no-ssl-certificate-verification --verbose --skip-ovftool-verification $(pwd)/tmp/vcsa.json"
  }
}

resource "local_sensitive_file" "devrc" {
  content = templatefile("./devrc.tpl", {
    nas_host        = equinix_metal_device.storage1.network.0.address
    esxi_host_1     = equinix_metal_device.esxi1.network.0.address
    esxi_host_1_pw  = equinix_metal_device.esxi1.root_password
    vsphere_host    = cidrhost("${equinix_metal_device.esxi1.network.0.address}/${equinix_metal_device.esxi1.network.0.cidr}", 3)
    private_network = "${equinix_metal_device.esxi1.network.2.address}/${equinix_metal_device.esxi1.network.2.cidr}"
  })
  filename = "./devrc"
}
