---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_host"
sidebar_current: "docs-vsphere-data-source-host"
description: |-
  A data source that can be used to get the ID of a host.
---

# vsphere\_host

The `vsphere_host` data source can be used to discover the ID of a vSphere
host. This can then be used with resources or data sources that require a host
managed object reference ID.

## Example Usage

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc1"
}

data "vsphere_host" "host" {
  name          = "esxi1"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}
```

## Argument Reference

The following arguments are supported:

* `datacenter_id` - (Required) The [managed object reference
  ID][docs-about-morefs] of a datacenter.
* `name` - (Optional) The name of the host. This can be a name or path. Can be
  omitted if there is only one host in your inventory.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

~> **NOTE:** When used against an ESXi host directly, this data source _always_
fetches the server's host object ID, regardless of what is entered into `name`.

## Attribute Reference

* `id` - The [managed objectID][docs-about-morefs] of this host.
* `resource_pool_id` - The [managed object ID][docs-about-morefs] of the host's
  root resource pool.

-> Note that the resource pool referenced by
[`resource_pool_id`](#resource_pool_id) is dependent on the target host's state
- if it's a standalone host, the resource pool will belong to the host only,
  however if it is a member of a cluster, the resource pool will be the root
  for the entire cluster.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider
