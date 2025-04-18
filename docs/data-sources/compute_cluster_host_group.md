---
subcategory: "Host and Cluster Management"
page_title: "VMware vSphere: vsphere_compute_cluster_host_group"
sidebar_current: "docs-vsphere-data-source-compute-cluster-host-group"
description: |-
  Provides a vSphere cluster host group data source. Returns attributes of a
  vSphere cluster host group.
---

# vsphere_compute_cluster_host_group

The `vsphere_compute_cluster_host_group` data source can be used to discover
the IDs ESXi hosts in a host group and return host group attributes to other
resources.

## Example Usage

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc-01"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "cluster-01"
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

data "vsphere_compute_cluster_host_group" "host_group" {
  name               = "hostgroup-01"
  compute_cluster_id = data.vsphere_compute_cluster.cluster.id
}

resource "vsphere_compute_cluster_vm_host_rule" "host_rule" {
  compute_cluster_id       = data.vsphere_compute_cluster.cluster.id
  name                     = "terraform-host-rule1"
  vm_group_name            = "vmgroup-01"
  affinity_host_group_name = data.vsphere_compute_cluster_host_group.host_group.name
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the host group.
* `compute_cluster_id` - (Required) The
  [managed object reference ID][docs-about-morefs] of the compute cluster for
  the host group.

## Attribute Reference

* `host_system_ids`: The [managed object reference ID][docs-about-morefs] of
  the ESXi hosts in the host group.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider
