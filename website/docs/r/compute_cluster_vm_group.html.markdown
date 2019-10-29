---
subcategory: "Host and Cluster Management"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_compute_cluster_vm_group"
sidebar_current: "docs-vsphere-resource-compute-cluster-vm-group"
description: |-
  Provides a VMware vSphere cluster virtual machine group. This can be used to manage groups of virtual machines for relevant rules in a cluster.
---

# vsphere\_compute\_cluster\_vm\_group

The `vsphere_compute_cluster_vm_group` resource can be used to manage groups of
virtual machines in a cluster, either created by the
[`vsphere_compute_cluster`][tf-vsphere-cluster-resource] resource or looked up
by the [`vsphere_compute_cluster`][tf-vsphere-cluster-data-source] data source.

[tf-vsphere-cluster-resource]: /docs/providers/vsphere/r/compute_cluster.html
[tf-vsphere-cluster-data-source]: /docs/providers/vsphere/d/compute_cluster.html

This resource mainly serves as an input to the
[`vsphere_compute_cluster_vm_dependency_rule`][tf-vsphere-cluster-vm-dependency-rule-resource]
and
[`vsphere_compute_cluster_vm_host_rule`][tf-vsphere-cluster-vm-host-rule-resource]
resources. See the individual resource documentation pages for more information.

[tf-vsphere-cluster-vm-dependency-rule-resource]: /docs/providers/vsphere/r/compute_cluster_vm_dependency_rule.html
[tf-vsphere-cluster-vm-host-rule-resource]: /docs/providers/vsphere/r/compute_cluster_vm_host_rule.html

~> **NOTE:** This resource requires vCenter and is not available on direct ESXi
connections.

~> **NOTE:** vSphere DRS requires a vSphere Enterprise Plus license.

## Example Usage

The example below creates two virtual machines in a cluster using the
[`vsphere_virtual_machine`][tf-vsphere-vm-resource] resource, creating the
virtual machine in the cluster looked up by the
[`vsphere_compute_cluster`][tf-vsphere-cluster-data-source] data source. It
then creates a group from these two virtual machines.

[tf-vsphere-vm-resource]: /docs/providers/vsphere/r/virtual_machine.html

```hcl
data "vsphere_datacenter" "dc" {
  name = "dc1"
}

data "vsphere_datastore" "datastore" {
  name          = "datastore1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "cluster1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "network1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  count            = 2
  name             = "terraform-test-${count.index}"
  resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }
}

resource "vsphere_compute_cluster_vm_group" "cluster_vm_group" {
  name                = "terraform-test-cluster-vm-group"
  compute_cluster_id  = "${data.vsphere_compute_cluster.cluster.id}"
  virtual_machine_ids = ["${vsphere_virtual_machine.vm.*.id}"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the VM group. This must be unique in the
  cluster. Forces a new resource if changed.
* `compute_cluster_id` - (Required) The [managed object reference
  ID][docs-about-morefs] of the cluster to put the group in.  Forces a new
  resource if changed.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

* `virtual_machine_ids` - (Required) The UUIDs of the virtual machines in this
  group.

~> **NOTE:** The namespace for cluster names on this resource (defined by the
[`name`](#name) argument) is shared with the
[`vsphere_compute_cluster_host_group`][tf-vsphere-cluster-host-group-resource]
resource. Make sure your names are unique across both resources.

[tf-vsphere-cluster-host-group-resource]: /docs/providers/vsphere/r/compute_cluster_host_group.html

## Attribute Reference

The only attribute this resource exports is the `id` of the resource, which is
a combination of the [managed object reference ID][docs-about-morefs] of the
cluster, and the name of the virtual machine group.

## Importing

An existing group can be [imported][docs-import] into this resource by
supplying both the path to the cluster, and the name of the VM group. If the
name or cluster is not found, or if the group is of a different type, an error
will be returned. An example is below:

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_compute_cluster_vm_group.cluster_vm_group \
  '{"compute_cluster_path": "/dc1/host/cluster1", \
  "name": "terraform-test-cluster-vm-group"}'
```
