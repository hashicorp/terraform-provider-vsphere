---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_compute_cluster"
sidebar_current: "docs-vsphere-data-source-compute-cluster"
description: |-
  Provides a vSphere cluster data source. This can be used to get the general attributes of a vSphere cluster.
---

# vsphere\_compute\_cluster

The `vsphere_compute_cluster` data source can be used to discover the ID of a
cluster in vSphere. This is useful to fetch the ID of a cluster that you want
to use for virtual machine placement via the
[`vsphere_virtual_machine`][docs-virtual-machine-resource] resource, allowing
you to specify the cluster's root resource pool directly versus using the alias
available through the [`vsphere_resource_pool`][docs-resource-pool-data-source]
data source.

[docs-virtual-machine-resource]: /docs/providers/vsphere/r/virtual_machine.html
[docs-resource-pool-data-source]: /docs/providers/vsphere/d/resource_pool.html

-> You may also wish to see the
[`vsphere_compute_cluster`][docs-compute-cluster-resource] resource for further
details about clusters or how to work with them in Terraform.

[docs-compute-cluster-resource]: /docs/providers/vsphere/r/compute_cluster.html

## Example Usage

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc1"
}

data "vsphere_compute_cluster" "compute_cluster" {
  name          = "compute-cluster1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name or absolute path to the cluster.
* `datacenter_id` - (Optional) The [managed object reference
  ID][docs-about-morefs] of the datacenter the cluster is located in.  This can
  be omitted if the search path used in `name` is an absolute path.  For
  default datacenters, use the id attribute from an empty `vsphere_datacenter`
  data source.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

## Attribute Reference

The following attributes are exported:

* `id`: The [managed object reference ID][docs-about-morefs] of the cluster.
* `resource_pool_id`: The [managed object reference ID][docs-about-morefs] of
  the root resource pool for the cluster.
