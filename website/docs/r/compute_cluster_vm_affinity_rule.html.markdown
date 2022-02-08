---
subcategory: "Host and Cluster Management"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_compute_cluster_vm_affinity_rule"
sidebar_current: "docs-vsphere-resource-compute-cluster-vm-affinity-rule"
description: |-
  Provides the VMware vSphere Distributed Resource Scheduler affinity rule resource.
---

# vsphere\_compute\_cluster\_vm\_affinity\_rule

The `vsphere_compute_cluster_vm_affinity_rule` resource can be used to
manage virtual machine affinity rules in a cluster, either created by the
[`vsphere_compute_cluster`][tf-vsphere-cluster-resource] resource or looked up
by the [`vsphere_compute_cluster`][tf-vsphere-cluster-data-source] data source.


[tf-vsphere-cluster-resource]: /docs/providers/vsphere/r/compute_cluster.html
[tf-vsphere-cluster-data-source]: /docs/providers/vsphere/d/compute_cluster.html

This rule can be used to tell a set of virtual machines to run together on the
same host within a cluster. When configured, DRS will make a best effort to
ensure that the virtual machines run on the same host, or prevent any operation
that would keep that from happening, depending on the value of the
[`mandatory`](#mandatory) flag.

-> An affinity rule can only be used to place virtual machines on the same
_non-specific_ hosts. It cannot be used to pin virtual machines to a host. 
To enable this capability, use the
[`vsphere_compute_cluster_vm_host_rule`][tf-vsphere-cluster-vm-host-rule-resource]
resource.

[tf-vsphere-cluster-vm-host-rule-resource]: /docs/providers/vsphere/r/compute_cluster_vm_host_rule.html

~> **NOTE:** This resource requires vCenter Server and is not available on
direct ESXi host connections.

~> **NOTE:** vSphere DRS requires a vSphere Enterprise Plus license.

## Example Usage

The following example creates two virtual machines in a cluster using the
[`vsphere_virtual_machine`][tf-vsphere-vm-resource] resource, creating the
virtual machines in the cluster looked up by the
[`vsphere_compute_cluster`][tf-vsphere-cluster-data-source] data source. It
then creates an affinity rule for these two virtual machines, ensuring they
will run on the same host whenever possible.

[tf-vsphere-vm-resource]: /docs/providers/vsphere/r/virtual_machine.html

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc-01"
}

data "vsphere_datastore" "datastore" {
  name          = "datastore-01"
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

data "vsphere_compute_cluster" "cluster" {
  name          = "cluster-01"
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

data "vsphere_network" "network" {
  name          = "VM Network"
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

resource "vsphere_virtual_machine" "vm" {
  count            = 2
  name             = "foo-${count.index}"
  resource_pool_id = data.vsphere_compute_cluster.cluster.resource_pool_id
  datastore_id     = data.vsphere_datastore.datastore.id

  num_cpus = 1
  memory   = 1024
  guest_id = "otherLinux64Guest"

  network_interface {
    network_id = data.vsphere_network.network.id
  }

  disk {
    label = "disk0"
    size  = 20
  }
}

resource "vsphere_compute_cluster_vm_affinity_rule" "vm_affinity_rule" {
  name                = "vm-affinity-rule"
  compute_cluster_id  = data.vsphere_compute_cluster.cluster.id
  virtual_machine_ids = [for k, v in vsphere_virtual_machine.vm : v.id]
}
```

The following example creates an affinity rule for a set of virtual machines
in the cluster by looking up the virtual machine UUIDs from the
[`vsphere_virtual_machine`][tf-vsphere-vm-data-source] data source. 

[tf-vsphere-vm-data-source]: /docs/providers/vsphere/d/virtual_machine.html

```hcl
locals {
  vms = [
    "foo-0",
    "foo-1"
  ]
}

data "vsphere_datacenter" "datacenter" {
  name = "dc-01"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "cluster-01"
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

data "vsphere_virtual_machine" "vms" {
  count         = length(local.vms)
  name          = local.vms[count.index]
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

resource "vsphere_compute_cluster_vm_affinity_rule" "vm_affinity_rule" {
  name                = "vm-affinity-rule"
  enabled             = true
  compute_cluster_id  = data.vsphere_compute_cluster.cluster.id
  virtual_machine_ids = data.vsphere_virtual_machine.vms[*].id
}
```

## Argument Reference

The following arguments are supported:

* `compute_cluster_id` - (Required) The [managed object reference
  ID][docs-about-morefs] of the cluster to put the group in.  Forces a new
  resource if changed.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

* `name` - (Required) The name of the rule. This must be unique in the cluster.
* `virtual_machine_ids` - (Required) The UUIDs of the virtual machines to run
  on the same host together.
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
terraform import vsphere_compute_cluster_vm_affinity_rule.vm_affinity_rule \
  '{"compute_cluster_path": "/dc-01/host/cluster-01", \
  "name": "vm-affinity-rule"}'
```