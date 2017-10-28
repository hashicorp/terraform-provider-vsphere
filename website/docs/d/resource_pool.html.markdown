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
  name          = "cluster1/Resources/resource-pool-1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}
```

### Specifying the default resource pool for a cluster

If you don't have any resource pools in your cluster, or you want to use the
parent resource pool for a cluster, you can just specify the `Resources` named
default in your path for the resource pool:

```
data "vsphere_resource_pool" "pool" {
  name          = "cluster1/Resources"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}
```

### Using with ESXi

On ESXi, you don't have to specify either attribute to use this data source. An
empty declaration will load the default resource pool.

```
data "vsphere_resource_pool" "pool" {}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the resource pool. This can be a name or
  path.
* `datacenter_id` - (Optional) The managed object reference ID of the
  datacenter the resource pool is located in. This can be omitted if the search
  path used in `name` is an absolute path, or if there is only one datacenter
  in the vSphere infrastructure.

## Attribute Reference

Currently, the only exported attribute from this data source is `id`, which
represents the ID of the resource pool that was looked up.
