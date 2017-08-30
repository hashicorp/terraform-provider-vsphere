---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_datacenter"
sidebar_current: "docs-vsphere-data-source-datacenter"
description: |-
  A data source that can be used to get the ID of a datacenter.
---

# vsphere\_datacenter

The `vsphere_datacenter` data source can be used to discover the ID of a
vSphere datacenter. This can then be used with resources or data sources that
require a datacenter, such as the [`vsphere_host`][data-source-vsphere-host]
data source.

[data-source-vsphere-host]: /docs/providers/vsphere/d/host.html

## Example Usage

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc1"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (String) The name of the datacenter. This can be a name or path.	If
  not provided, the default datacenter is used.

~> **NOTE:** When used against ESXi, this data source _always_ fetches the
server's "default" datacenter, which is a special datacenter unrelated to the
default datacenter that exists in vCenter. Hence, the `name` attribute is
completely ignored.

## Attribute Reference

The only exported attribute is `id`, which is the managed object ID of this
datacenter.
