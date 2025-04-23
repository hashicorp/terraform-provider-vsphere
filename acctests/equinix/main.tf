# © Broadcom. All Rights Reserved.
# The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
# SPDX-License-Identifier: MPL-2.0

variable "PACKET_AUTH" {
  sensitive = true
}

variable "PACKET_PROJECT" {}

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

variable "LAB_PREFIX" {
  default = ""
}

variable "DOMAIN" {
  default = "test.local"
}

provider "equinix" {
  auth_token = var.PACKET_AUTH
}

resource "tls_private_key" "gh-actions-ssh" {
  algorithm = "RSA" # for some reason ED25519 does not work
  rsa_bits  = 4096
}

resource "local_sensitive_file" "gh-actions-ssh" {
  content         = tls_private_key.gh-actions-ssh.private_key_openssh
  filename        = "./gh-actions-ssh"
  file_permission = "0600"
}

resource "equinix_metal_project_ssh_key" "gh-actions-ssh" {
  name       = "gh-actions-ssh"
  public_key = tls_private_key.gh-actions-ssh.public_key_openssh
  project_id = var.PACKET_PROJECT
}

resource "equinix_metal_device" "esxi1" {
  hostname            = "${var.LAB_PREFIX}e.${var.DOMAIN}"
  plan                = var.ESXI_PLAN
  metro               = var.PACKET_FACILITY
  operating_system    = var.ESXI_VERSION
  billing_cycle       = "hourly"
  project_ssh_key_ids = [equinix_metal_project_ssh_key.gh-actions-ssh.id]
  project_id          = var.PACKET_PROJECT
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

resource "random_password" "admin" {
  length      = 12
  min_upper   = 1
  min_lower   = 1
  min_numeric = 1
  min_special = 1
}

locals {
  vcenter_fqdn = "vcenter.${var.DOMAIN}"
}

resource "local_sensitive_file" "vcsa_template" {
  depends_on = [equinix_metal_port_vlan_attachment.esxi1]

  content = templatefile("${path.cwd}/vcsa_deploy.json", {
    hostname       = equinix_metal_device.esxi1.network.0.address
    password       = equinix_metal_device.esxi1.root_password
    ip_address     = cidrhost("${equinix_metal_device.esxi1.network.0.address}/${equinix_metal_device.esxi1.network.0.cidr}", 3)
    ip_prefix      = equinix_metal_device.esxi1.network.0.cidr
    gateway        = cidrhost("${equinix_metal_device.esxi1.network.0.address}/${equinix_metal_device.esxi1.network.0.cidr}", 1)
    vcenter_fqdn   = local.vcenter_fqdn
    admin_password = random_password.admin.result
  })
  filename = "./tmp/vcsa.json"
  provisioner "local-exec" {
    command = "sleep 300; TERM=xterm-256color ${var.VCSA_DEPLOY_PATH} install --accept-eula --acknowledge-ceip --no-ssl-certificate-verification --verbose --skip-ovftool-verification $(pwd)/tmp/vcsa.json"
  }
}

resource "local_sensitive_file" "devrc" {
  content = templatefile("./devrc.tpl", {
    esxi_host_1     = equinix_metal_device.esxi1.network.0.address
    esxi_host_1_pw  = equinix_metal_device.esxi1.root_password
    vsphere_host    = cidrhost("${equinix_metal_device.esxi1.network.0.address}/${equinix_metal_device.esxi1.network.0.cidr}", 3)
    public_network  = "${cidrhost("${equinix_metal_device.esxi1.network.0.address}/${equinix_metal_device.esxi1.network.0.cidr}", 0)}/${equinix_metal_device.esxi1.network.0.cidr}"
    private_network = "${cidrhost("${equinix_metal_device.esxi1.network.2.address}/${equinix_metal_device.esxi1.network.2.cidr}", 0)}/${equinix_metal_device.esxi1.network.2.cidr}"
    admin_user      = "administrator@${local.vcenter_fqdn}"
    admin_password  = random_password.admin.result
    priv_key        = "${path.cwd}/gh-actions-ssh"
  })
  filename = "./devrc"
}
