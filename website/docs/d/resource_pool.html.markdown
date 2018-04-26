---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_resource_pool"
sidebar_current: "docs-vsphere-data-source-resource-pool"
description: |-
  Provides a vSphere resource pool data source. This can be used to get the general attributes of a vSphere resource pool.
---

# vsphere\_resource\_pool

The `vsphere_resource_pool` data source can be used to discover the ID of a
resource pool in vSphere. This is useful to fetch the ID of a resource pool
that you want to use to create virtual machines in using the
[`vsphere_virtual_machine`][docs-virtual-machine-resource] resource. 

[docs-virtual-machine-resource]: /docs/providers/vsphere/r/virtual_machine.html

## Example Usage

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc1"
}

data "vsphere_resource_pool" "pool" {
  name          = "resource-pool-1"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}
```

### Specifying the root resource pool for a standalone host

-> **NOTE:** Fetching the root resource pool for a cluster can now be done
directly via the [`vsphere_compute_cluster`][docs-compute-cluster-data-source]
data source.

[docs-compute-cluster-data-source]: /docs/providers/vsphere/d/compute_cluster.html

All compute resources in vSphere (clusters, standalone hosts, and standalone
ESXi) have a resource pool, even if one has not been explicitly created. This
resource pool is referred to as the _root resource pool_ and can be looked up
by specifying the path as per the example below:

```
data "vsphere_resource_pool" "pool" {
  name          = "esxi1/Resources"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}
```

For more information on the root resource pool, see [Managing Resource
Pools][vmware-docs-resource-pools] in the vSphere documentation.

[vmware-docs-resource-pools]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.resmgmt.doc/GUID-60077B40-66FF-4625-934A-641703ED7601.html

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The name of the resource pool. This can be a name or
  path. This is required when using vCenter.
* `datacenter_id` - (Optional) The [managed object reference
  ID][docs-about-morefs] of the datacenter the resource pool is located in.
  This can be omitted if the search path used in `name` is an absolute path.
  For default datacenters, use the id attribute from an empty
  `vsphere_datacenter` data source.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

~> **Note when using with standalone ESXi:** When using ESXi without vCenter,
you don't have to specify either attribute to use this data source. An empty
declaration will load the host's root resource pool.

## Attribute Reference

Currently, the only exported attribute from this data source is `id`, which
represents the ID of the resource pool that was looked up.
