---
subcategory: "Host and Cluster Management"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_compute_cluster_vm_host_rule"
sidebar_current: "docs-vsphere-resource-compute-cluster-vm-host-rule"
description: |-
  Provides a VMware vSphere cluster VM/host rule. This can be used to manage VM-to-host affinity and anti-affinity rules.
---

# vsphere\_compute\_cluster\_vm\_host\_rule

The `vsphere_compute_cluster_vm_host_rule` resource can be used to manage
VM-to-host rules in a cluster, either created by the
[`vsphere_compute_cluster`][tf-vsphere-cluster-resource] resource or looked up
by the [`vsphere_compute_cluster`][tf-vsphere-cluster-data-source] data source.

[tf-vsphere-cluster-resource]: /docs/providers/vsphere/r/compute_cluster.html
[tf-vsphere-cluster-data-source]: /docs/providers/vsphere/d/compute_cluster.html

This resource can create both _affinity rules_, where virtual machines run on
specified hosts, or _anti-affinity_ rules, where virtual machines run on hosts
outside of the ones specified in the rule. Virtual machines and hosts are
supplied via groups, which can be managed via the
[`vsphere_compute_cluster_vm_group`][tf-vsphere-cluster-vm-group-resource] and
[`vsphere_compute_cluster_host_group`][tf-vsphere-cluster-host-group-resource]
resources.

[tf-vsphere-cluster-vm-group-resource]: /docs/providers/vsphere/r/compute_cluster_vm_group.html
[tf-vsphere-cluster-host-group-resource]: /docs/providers/vsphere/r/compute_cluster_host_group.html

~> **NOTE:** This resource requires vCenter and is not available on direct ESXi
connections.

~> **NOTE:** vSphere DRS requires a vSphere Enterprise Plus license.

## Example Usage

The example below creates a virtual machine in a cluster using the
[`vsphere_virtual_machine`][tf-vsphere-vm-resource] resource in a cluster
looked up by the [`vsphere_compute_cluster`][tf-vsphere-cluster-data-source]
data source. It then creates a group with this virtual machine. It also creates
a host group off of the host looked up via the
[`vsphere_host`][tf-vsphere-host-data-source] data source. Finally, this
virtual machine is configured to run specifically on that host via a
`vsphere_compute_cluster_vm_host_rule` resource.

[tf-vsphere-vm-resource]: /docs/providers/vsphere/r/virtual_machine.html
[tf-vsphere-host-data-source]: /docs/providers/vsphere/d/host.html

-> Note how [`vm_group_name`](#vm_group_name) and
[`affinity_host_group_name`](#affinity_host_group_name) are sourced off of the
`name` attributes from the
[`vsphere_compute_cluster_vm_group`][tf-vsphere-cluster-vm-group-resource] and
[`vsphere_compute_cluster_host_group`][tf-vsphere-cluster-host-group-resource]
resources. This is to ensure that the rule is not created before the groups
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

data "vsphere_host" "host" {
  name          = "esxi1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "network1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
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
  virtual_machine_ids = ["${vsphere_virtual_machine.vm.id}"]
}

resource "vsphere_compute_cluster_host_group" "cluster_host_group" {
  name               = "terraform-test-cluster-vm-group"
  compute_cluster_id = "${data.vsphere_compute_cluster.cluster.id}"
  host_system_ids    = ["${data.vsphere_host.host.id}"]
}

resource "vsphere_compute_cluster_vm_host_rule" "cluster_vm_host_rule" {
  compute_cluster_id       = "${data.vsphere_compute_cluster.cluster.id}"
  name                     = "terraform-test-cluster-vm-host-rule"
  vm_group_name            = "${vsphere_compute_cluster_vm_group.cluster_vm_group.name}"
  affinity_host_group_name = "${vsphere_compute_cluster_host_group.cluster_host_group.name}"
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
* `vm_group_name` - (Required) The name of the virtual machine group to use
  with this rule.
* `affinity_host_group_name` - (Optional) When this field is used, the virtual
  machines defined in [`vm_group_name`](#vm_group_name) will be run on the
  hosts defined in this host group.
* `anti_affinity_host_group_name` - (Optional) When this field is used, the
  virtual machines defined in [`vm_group_name`](#vm_group_name) will _not_ be
  run on the hosts defined in this host group.
* `enabled` - (Optional) Enable this rule in the cluster. Default: `true`.
* `mandatory` - (Optional) When this value is `true`, prevents any virtual
  machine operations that may violate this rule. Default: `false`.

~> **NOTE:** One of [`affinity_host_group_name`](#affinity_host_group_name) or
[`anti_affinity_host_group_name`](#anti_affinity_host_group_name) must be
defined, but not both.

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
terraform import vsphere_compute_cluster_vm_host_rule.cluster_vm_host_rule \
  '{"compute_cluster_path": "/dc1/host/cluster1", \
  "name": "terraform-test-cluster-vm-host-rule"}'
```
