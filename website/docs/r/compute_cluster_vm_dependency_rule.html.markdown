---
subcategory: "Host and Cluster Management"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_compute_cluster_vm_dependency_rule"
sidebar_current: "docs-vsphere-resource-compute-cluster-vm-dependency-rule"
description: |-
  Provides a VMware vSphere cluster VM dependency rule. This can be used to manage VM dependency rules for vSphere HA.
---

# vsphere\_compute\_cluster\_vm\_dependency\_rule

The `vsphere_compute_cluster_vm_dependency_rule` resource can be used to manage
VM dependency rules in a cluster, either created by the
[`vsphere_compute_cluster`][tf-vsphere-cluster-resource] resource or looked up
by the [`vsphere_compute_cluster`][tf-vsphere-cluster-data-source] data source.

[tf-vsphere-cluster-resource]: /docs/providers/vsphere/r/compute_cluster.html
[tf-vsphere-cluster-data-source]: /docs/providers/vsphere/d/compute_cluster.html

A virtual machine dependency rule applies to vSphere HA, and allows
user-defined startup orders for virtual machines in the case of host failure.
Virtual machines are supplied via groups, which can be managed via the
[`vsphere_compute_cluster_vm_group`][tf-vsphere-cluster-vm-group-resource]
resource.

[tf-vsphere-cluster-vm-group-resource]: /docs/providers/vsphere/r/compute_cluster_vm_group.html

~> **NOTE:** This resource requires vCenter and is not available on direct ESXi
connections.

## Example Usage

The example below creates two virtual machine in a cluster using the
[`vsphere_virtual_machine`][tf-vsphere-vm-resource] resource in a cluster
looked up by the [`vsphere_compute_cluster`][tf-vsphere-cluster-data-source]
data source. It then creates a group with this virtual machine. Two groups are created, each with one of the created VMs. Finally, a rule is created to ensure that `vm1` starts before `vm2`.

[tf-vsphere-vm-resource]: /docs/providers/vsphere/r/virtual_machine.html

-> Note how [`dependency_vm_group_name`](#dependency_vm_group_name) and
[`vm_group_name`](#vm_group_name) are sourced off of the `name` attributes from
the [`vsphere_compute_cluster_vm_group`][tf-vsphere-cluster-vm-group-resource]
resource. This is to ensure that the rule is not created before the groups
exist, which may not possibly happen in the event that the names came from a
"static" source such as a variable.

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

resource "vsphere_virtual_machine" "vm1" {
  name             = "terraform-test1"
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

resource "vsphere_virtual_machine" "vm2" {
  name             = "terraform-test2"
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

resource "vsphere_compute_cluster_vm_group" "cluster_vm_group1" {
  name                = "terraform-test-cluster-vm-group1"
  compute_cluster_id  = "${data.vsphere_compute_cluster.cluster.id}"
  virtual_machine_ids = ["${vsphere_virtual_machine.vm1.id}"]
}

resource "vsphere_compute_cluster_vm_group" "cluster_vm_group2" {
  name                = "terraform-test-cluster-vm-group2"
  compute_cluster_id  = "${data.vsphere_compute_cluster.cluster.id}"
  virtual_machine_ids = ["${vsphere_virtual_machine.vm2.id}"]
}

resource "vsphere_compute_cluster_vm_dependency_rule" "cluster_vm_dependency_rule" {
  compute_cluster_id       = "${data.vsphere_compute_cluster.cluster.id}"
  name                     = "terraform-test-cluster-vm-dependency-rule"
  dependency_vm_group_name = "${vsphere_compute_cluster_vm_group.cluster_vm_group1.name}"
  vm_group_name            = "${vsphere_compute_cluster_vm_group.cluster_vm_group2.name}"
}
```

## Argument Reference

The following arguments are supported:

* `compute_cluster_id` - (Required) The [managed object reference
  ID][docs-about-morefs] of the cluster to put the group in.  Forces a new
  resource if changed.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

* `name` - (Required) The name of the rule. This must be unique in the
  cluster.
* `dependency_vm_group_name` - (Required) The name of the VM group that this
  rule depends on. The VMs defined in the group specified by
  [`vm_group_name`](#vm_group_name) will not be started until the VMs in this
  group are started.
* `vm_group_name` - (Required) The name of the VM group that is the subject of
  this rule. The VMs defined in this group will not be started until the VMs in
  the group specified by
  [`dependency_vm_group_name`](#dependency_vm_group_name) are started.
* `enabled` - (Optional) Enable this rule in the cluster. Default: `true`.
* `mandatory` - (Optional) When this value is `true`, prevents any virtual
  machine operations that may violate this rule. Default: `false`.

~> **NOTE:** The namespace for rule names on this resource (defined by the
[`name`](#name) argument) is shared with all rules in the cluster - consider
this when naming your rules.

## Attribute Reference

The only attribute this resource exports is the `id` of the resource, which is
a combination of the [managed object reference ID][docs-about-morefs] of the
cluster, and the rule's key within the cluster configuration.

## Importing

An existing rule can be [imported][docs-import] into this resource by supplying
both the path to the cluster, and the name the rule. If the name or cluster is
not found, or if the rule is of a different type, an error will be returned. An
example is below:

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_compute_cluster_vm_dependency_rule.cluster_vm_dependency_rule \
  '{"compute_cluster_path": "/dc1/host/cluster1", \
  "name": "terraform-test-cluster-vm-dependency-rule"}'
```
