// example_datacenter looks up the datacenter where all resources will be
// placed.
data "vsphere_datacenter" "example_datacenter" {
  name = "${data.vsphere_datacenter.rootdc1.name}"
}

// example_hosts looks up the IDs for the hosts that will be used during
// datastore and distributed virtual switch creation, in addition to defining
// where each virtual machine will be placed. One virtual machine per host will
// be created.
data "vsphere_host" "example_hosts" {
  count         = "${length(var.esxi_hosts)}"
  name          = "${var.esxi_hosts[count.index]}"
  datacenter_id = "${data.vsphere_datacenter.example_datacenter.id}"
}

// example_resource_pool looks up the resource pool to place the virtual machines in.
data "vsphere_resource_pool" "example_resource_pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.example_datacenter.id}"
}

// example_template looks up the template to create the virtual machines as.
data "vsphere_virtual_machine" "example_template" {
  name          = "${var.template_name}"
  datacenter_id = "${data.vsphere_datacenter.example_datacenter.id}"
}
