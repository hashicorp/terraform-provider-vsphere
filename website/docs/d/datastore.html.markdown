---
subcategory: "Storage"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_datastore"
sidebar_current: "docs-vsphere-data-source-datastore"
description: |-
  Provides a data source to return the ID of a vSphere datastore object.
---

# vsphere\_datastore

The `vsphere_datastore` data source can be used to discover the ID of a
vSphere datastore object. This can then be used with resources or data sources
that require a datastore. For example, to create virtual machines in using the
[`vsphere_virtual_machine`][docs-virtual-machine-resource] resource.

[docs-virtual-machine-resource]: /docs/providers/vsphere/r/virtual_machine.html

## Example Usage

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc-01"
}

data "vsphere_datastore" "datastore" {
  name          = "datastore-01"
  datacenter_id = data.vsphere_datacenter.datacenter.id
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the datastore. This can be a name or path.
* `datacenter_id` - (Optional) The [managed object reference ID][docs-about-morefs]
  of the datacenter the datastore is located in. This can be omitted if the
  search path used in `name` is an absolute path. For default datacenters, use
  the `id` attribute from an empty `vsphere_datacenter` data source.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

## Attribute Reference

The only exported attribute from this data source is `id`, which represents the
ID of the datastore.
