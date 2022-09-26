---
subcategory: "Host and Cluster Management"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_host"
sidebar_current: "docs-vsphere-data-source-host"
description: |-
  A data source that can be used to return the attributes of an ESXi host.
---

# vsphere\_host

The `vsphere_host` data source can be used to discover the ID of an ESXi host.
This can then be used with resources or data sources that require an ESX
host's [managed object reference ID][docs-about-morefs].

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

## Example Usage

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc-01"
}

data "vsphere_host" "host" {
  name          = "esxi-01.example.com"
  datacenter_id = data.vsphere_datacenter.datacenter.id
}
```

## Argument Reference

The following arguments are supported:

* `datacenter_id` - (Required) The [managed object reference ID][docs-about-morefs]
  of a vSphere datacenter object.
* `name` - (Optional) The name of the ESXI host. This can be a name or path.
  Can be omitted if there is only one host in your inventory.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

~> **NOTE:** When used against an ESXi host directly, this data source _always_
returns the ESXi host's object ID, regardless of what is entered into `name`.

## Attribute Reference

* `id` - The [managed objectID][docs-about-morefs] of the ESXi host.
* `resource_pool_id` - The [managed object ID][docs-about-morefs] of the ESXi
  host's root resource pool.

-> Note that the resource pool referenced by [`resource_pool_id`](#resource_pool_id)
  is dependent on the ESXi host's state. If it is a standalone ESXi host, the
  resource pool will belong to the host only; however, if it is a member of a
  cluster, the resource pool will be the root for the cluster.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider
