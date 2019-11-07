---
subcategory: "Storage"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_datastore_cluster_vm_anti_affinity_rule"
sidebar_current: "docs-vsphere-resource-storage-cluster-datastore-vm-anti-affinity-rule"
description: |-
  Provides a VMware vSphere datastore cluster virtual machine anti-affinity rule. This can be used to manage rules to tell virtual machines to run on separate datastores.
---

# vsphere\_datastore\_cluster\_vm\_anti\_affinity\_rule

The `vsphere_datastore_cluster_vm_anti_affinity_rule` resource can be used to
manage VM anti-affinity rules in a datastore cluster, either created by the
[`vsphere_datastore_cluster`][tf-vsphere-datastore-cluster-resource] resource or looked up
by the [`vsphere_datastore_cluster`][tf-vsphere-datastore-cluster-data-source] data source.

[tf-vsphere-datastore-cluster-resource]: /docs/providers/vsphere/r/datastore_cluster.html
[tf-vsphere-datastore-cluster-data-source]: /docs/providers/vsphere/d/datastore_cluster.html

This rule can be used to tell a set to virtual machines to run on different
datastores within a cluster, useful for preventing single points of failure in
application cluster scenarios. When configured, Storage DRS will make a best effort to
ensure that the virtual machines run on different datastores, or prevent any
operation that would keep that from happening, depending on the value of the
[`mandatory`](#mandatory) flag.

~> **NOTE:** This resource requires vCenter and is not available on direct ESXi
connections.

~> **NOTE:** Storage DRS requires a vSphere Enterprise Plus license.

## Example Usage

The example below creates two virtual machines in a cluster using the
[`vsphere_virtual_machine`][tf-vsphere-vm-resource] resource, creating the
virtual machines in the datastore cluster looked up by the
[`vsphere_datastore_cluster`][tf-vsphere-datastore-cluster-data-source] data
source. It then creates an anti-affinity rule for these two virtual machines,
ensuring they will run on different datastores whenever possible.

[tf-vsphere-vm-resource]: /docs/providers/vsphere/r/virtual_machine.html

```hcl
data "vsphere_datacenter" "dc" {
  name = "dc1"
}

data "vsphere_datastore_cluster" "datastore_cluster" {
  name          = "datastore-cluster1"
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
  count                = 2
  name                 = "terraform-test-${count.index}"
  resource_pool_id     = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
  datastore_cluster_id = "${data.vsphere_datastore_cluster.datastore_cluster.id}"

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

resource "vsphere_datastore_cluster_vm_anti_affinity_rule" "cluster_vm_anti_affinity_rule" {
  name                 = "terraform-test-datastore-cluster-vm-anti-affinity-rule"
  datastore_cluster_id = "${data.vsphere_datastore_cluster.datastore_cluster.id}"
  virtual_machine_ids  = ["${vsphere_virtual_machine.vm.*.id}"]
}
```

## Argument Reference

The following arguments are supported:

* `datastore_cluster_id` - (Required) The [managed object reference
  ID][docs-about-morefs] of the datastore cluster to put the group in.  Forces
  a new resource if changed.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

* `name` - (Required) The name of the rule. This must be unique in the cluster.
* `virtual_machine_ids` - (Required) The UUIDs of the virtual machines to run
  on different datastores from each other.

~> **NOTE:** The minimum length of `virtual_machine_ids` is 2, and due to
current limitations in Terraform Core, the value is currently checked during
the apply phase, not the validation or plan phases. Ensure proper length of
this value to prevent failures mid-apply.

* `enabled` - (Optional) Enable this rule in the cluster. Default: `true`.
* `mandatory` - (Optional) When this value is `true`, prevents any virtual
  machine operations that may violate this rule. Default: `false`.

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
terraform import vsphere_datastore_cluster_vm_anti_affinity_rule.cluster_vm_anti_affinity_rule \
  '{"compute_cluster_path": "/dc1/datastore/cluster1", \
  "name": "terraform-test-datastore-cluster-vm-anti-affinity-rule"}'
```
