---
subcategory: "Inventory"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_datacenter"
sidebar_current: "docs-vsphere-data-source-datacenter"
description: |-
  Provides a data source to return the ID of a vSphere datacenter object.
---

# vsphere\_datacenter

The `vsphere_datacenter` data source can be used to discover the ID of a vSphere
datacenter object. This can then be used with resources or data sources that
require a datacenter, such as the [`vsphere_host`][data-source-vsphere-host]
data source.

[data-source-vsphere-host]: /docs/providers/vsphere/d/host.html

## Example Usage

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc-01"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The name of the datacenter. This can be a name or path.
  Can be omitted if there is only one datacenter in the inventory.

~> **NOTE:** When used with an ESXi host, this data source _always_ returns the
host's "default" datacenter, which is a special datacenter name unrelated to the
datacenters that exist in the vSphere inventory when managed by a vCenter Server
instance. Hence, the `name` attribute is completely ignored.

## Attribute Reference

The following attributes are exported:

* `id` - The [managed objectID][docs-about-morefs] of the vSphere datacenter object.
* `virtual_machines` - List of all virtual machines included in the vSphere datacenter object.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider
