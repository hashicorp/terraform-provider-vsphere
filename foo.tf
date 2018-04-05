data "vsphere_datacenter" "dc" {
  name = "dc1"
}

data "vsphere_datastore_cluster" "datastore_cluster" {
  name          = "datastore-cluster1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_datastore" "member_datastore" {
  name          = "datastore-cluster1-member1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "cluster1/Resources"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "public"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name                 = "terraform-test"
  resource_pool_id     = "${data.vsphere_resource_pool.pool.id}"
  datastore_id         = "${data.vsphere_datastore.member_datastore.id}"
  datastore_cluster_id = "${data.vsphere_datastore_cluster.datastore_cluster.id}"

  sdrs_enabled           = false
  sdrs_automation_level  = "automated"
  sdrs_intra_vm_affinity = false

  num_cpus = 2
  memory   = 1024
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }
}

// resource "vsphere_storage_drs_vm_override" "drs_vm_override" {
//   datastore_cluster_id = "${data.vsphere_datastore_cluster.datastore_cluster.id}"
//   virtual_machine_id   = "${vsphere_virtual_machine.vm.id}"
//   sdrs_enabled         = false
// }

