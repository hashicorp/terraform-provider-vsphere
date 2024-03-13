---
subcategory: "Storage"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_datastore_stats"
sidebar_current: "docs-vsphere-data-source-datastore-stats"
description: |-
  Provides a data source to return the usage stats for all vSphere datastore objects
  in a datacenter.
---

# vsphere_datastore_stats

The `vsphere_datastore_stats` data source can be used to retrieve the usage stats
of all vSphere datastore objects in a datacenter. This can then be used as a
standalone datasource to get information required as input to other data sources.

## Example Usage

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc-01"
}

data "vsphere_datastore_stats" "datastore_stats" {
  datacenter_id = data.vsphere_datacenter.datacenter.id
}
```

A usefull example of this datasource would be to determine the
datastore with the most free space. For example, in addition to
the above:

Create an `outputs.tf` like that:

```hcl
output "max_free_space_name" {
  value = local.max_free_space_name
}

output "max_free_space" {
  value = local.max_free_space
}
```

and a `locals.tf` like that:

```hcl
locals {
  free_space_values   = { for k, v in data.vsphere_datastore_stats.datastore_stats.free_space : k => tonumber(v) }
  filtered_values     = { for k, v in local.free_space_values : k => tonumber(v) if v != null }
  numeric_values      = [for v in values(local.filtered_values) : tonumber(v)]
  max_free_space      = max(local.numeric_values...)
  max_free_space_name = [for k, v in local.filtered_values : k if v == local.max_free_space][0]
}
```

This way you can get the storage object with the most
space available.

## Argument Reference

The following arguments are supported:

- `datacenter_id` - (Required) The [managed object reference ID][docs-about-morefs]
  of the datacenter the datastores are located in. For default datacenters, use
  the `id` attribute from an empty `vsphere_datacenter` data source.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

## Attribute Reference

The following attributes are exported:

- `datacenter_id` - The [managed object reference ID][docs-about-morefs]
  of the datacenter the datastores are located in.
- `free_space` - A mapping of the free space for each datastore in the
  datacenter, where the name of the datastore is used as key and the free
  space as value.
- `capacity` - A mapping of the capacity for all datastore in the datacenter
  , where the name of the datastore is used as key and the capacity as value.
