terraform {
  required_providers {
    vsphere = {
      source = "hashicorp/vsphere"
    }
  }
}

provider "vsphere" {
  user                 = var.vcenter_username
  password             = var.vcenter_password
  vsphere_server       = var.vcenter_server
  allow_unverified_ssl = true
}

resource "vsphere_datacenter" "dc" {
  name = "acc-test-dc"
}

resource "vsphere_host" "host1" {
  datacenter = vsphere_datacenter.dc.moid
  hostname = var.hosts[0].hostname
  username =  var.hosts[0].username
  password =  var.hosts[0].password
}

resource "vsphere_host" "host2" {
  datacenter = vsphere_datacenter.dc.moid
  hostname = var.hosts[1].hostname
  username =  var.hosts[1].username
  password =  var.hosts[1].password
}

resource "vsphere_compute_cluster" "cluster" {
  datacenter_id = vsphere_datacenter.dc.moid
  name          = "acc-test-cluster"

  host_system_ids = [
    vsphere_host.host1.id,
    vsphere_host.host2.id
  ]
}