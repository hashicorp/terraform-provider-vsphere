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

provider "metal" {
  auth_token = var.PACKET_AUTH
}

resource "metal_device" "esxi1" {
  hostname         = "${var.LAB_PREFIX}esxi1.vspheretest.internal"
  plan             = var.ESXI_PLAN
  metro            = var.PACKET_FACILITY
  operating_system = var.ESXI_VERSION
  billing_cycle    = "hourly"
  project_id       = var.PACKET_PROJECT
}

resource "metal_device_network_type" "esxi1" {
  device_id = metal_device.esxi1.id
  type      = "hybrid"
}

resource "metal_device" "esxi2" {
  hostname         = "${var.LAB_PREFIX}esxi2.vspheretest.internal"
  plan             = var.ESXI_PLAN
  metro            = var.PACKET_FACILITY
  operating_system = var.ESXI_VERSION
  billing_cycle    = "hourly"
  project_id       = var.PACKET_PROJECT
}

resource "metal_device_network_type" "esxi2" {
  device_id = metal_device.esxi2.id
  type      = "hybrid"
}

resource "metal_device" "esxi3" {
  hostname         = "${var.LAB_PREFIX}esxi3.vspheretest.internal"
  plan             = var.ESXI_PLAN
  metro            = var.PACKET_FACILITY
  operating_system = var.ESXI_VERSION
  billing_cycle    = "hourly"
  project_id       = var.PACKET_PROJECT
}

resource "metal_device_network_type" "esxi3" {
  device_id = metal_device.esxi3.id
  type      = "hybrid"
}

resource "metal_device" "esxi4" {
  hostname         = "${var.LAB_PREFIX}esxi4.vspheretest.internal"
  plan             = var.ESXI_PLAN
  metro            = var.PACKET_FACILITY
  operating_system = var.ESXI_VERSION
  billing_cycle    = "hourly"
  project_id       = var.PACKET_PROJECT
}

resource "metal_device_network_type" "esxi4" {
  device_id = metal_device.esxi4.id
  type      = "hybrid"
}

resource "metal_device" "storage1" {
  hostname         = "${var.LAB_PREFIX}storage1.vspheretest.internal"
  plan             = var.STORAGE_PLAN
  metro            = var.PACKET_FACILITY
  operating_system = "ubuntu_20_04"
  billing_cycle    = "hourly"
  project_id       = var.PACKET_PROJECT
  provisioner "remote-exec" {
    inline = [
      "mkdir -p /nfs/ds1 /nfs/ds2 /nfs/ds2 /nfs/ds3",
      "apt-get update",
      "apt-get install nfs-common nfs-kernel-server -y",
      "echo \"/nfs *(rw,no_root_squash)\" > /etc/exports",
      "echo \"/nfs/ds1 *(rw,no_root_squash)\" >> /etc/exports",
      "echo \"/nfs/ds2 *(rw,no_root_squash)\" >> /etc/exports",
      "echo \"/nfs/ds3 *(rw,no_root_squash)\" >> /etc/exports",
      "exportfs -a",
    ]
    connection {
      host        = metal_device.storage1.network.0.address
      private_key = file(var.PRIV_KEY)
    }
  }
}

resource "metal_vlan" "vmvlan" {
  metro      = var.PACKET_FACILITY
  project_id = var.PACKET_PROJECT
}

resource "metal_port_vlan_attachment" "vmvlan_esxi1" {
  device_id = metal_device.esxi1.id
  port_name = "eth1"
  vlan_vnid = metal_vlan.vmvlan.vxlan
}

resource "metal_port_vlan_attachment" "vmvlan_esxi2" {
  device_id = metal_device.esxi2.id
  port_name = "eth1"
  vlan_vnid = metal_vlan.vmvlan.vxlan
}

resource "metal_port_vlan_attachment" "vmvlan_esxi3" {
  device_id = metal_device.esxi3.id
  port_name = "eth1"
  vlan_vnid = metal_vlan.vmvlan.vxlan
}

resource "metal_port_vlan_attachment" "vmvlan_esxi4" {
  device_id = metal_device.esxi4.id
  port_name = "eth1"
  vlan_vnid = metal_vlan.vmvlan.vxlan
}

resource "local_file" "vcsa_template" {
  content = templatefile("${path.cwd}/vcsa_deploy.json", {
    hostname       = metal_device.esxi1.network.0.address
    password       = metal_device.esxi1.root_password
    ip_address     = cidrhost("${metal_device.esxi1.network.0.address}/${metal_device.esxi1.network.0.cidr}", 3)
    ip_prefix      = metal_device.esxi1.network.0.cidr
    gateway        = cidrhost("${metal_device.esxi1.network.0.address}/${metal_device.esxi1.network.0.cidr}", 1)
    vcenter_fqdn   = "vcenter.vspheretest.internal"
    admin_password = "Password123!"
  })
  filename = "./tmp/vcsa.json"
  provisioner "local-exec" {
    command = "sleep 290; echo five more; sleep 290; TERM=xterm-256color ${var.VCSA_DEPLOY_PATH} install --accept-eula --acknowledge-ceip --no-ssl-certificate-verification --verbose --skip-ovftool-verification $(pwd)/tmp/vcsa.json"
  }
}

output "ip" {
  value = cidrhost("${metal_device.esxi1.network.0.address}/${metal_device.esxi1.network.0.cidr}", 3)
}

resource "local_file" "devrc" {
  sensitive_content = templatefile("./devrc.tpl", {
    nas_host       = metal_device.storage1.network.0.address
    esxi_host_1    = metal_device.esxi1.network.0.address
    esxi_host_1_pw = metal_device.esxi1.root_password
    esxi_host_2    = metal_device.esxi2.network.0.address
    esxi_host_2_pw = metal_device.esxi2.root_password
    esxi_host_3    = metal_device.esxi3.network.0.address
    esxi_host_3_pw = metal_device.esxi3.root_password
    esxi_host_4    = metal_device.esxi4.network.0.address
    esxi_host_4_pw = metal_device.esxi4.root_password
    vsphere_host   = cidrhost("${metal_device.esxi1.network.0.address}/${metal_device.esxi1.network.0.cidr}", 3)
  })
  filename = "./devrc"
}
