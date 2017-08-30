---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_datacenter"
sidebar_current: "docs-vsphere-data-source-datacenter"
description: |-
  A data source that can be used to get the ID of a datacenter.
---

# vsphere\_datacenter

The `vsphere_datacenter` data source can be used to discover the ID of a
vSphere datacenter. It can also be used to fetch the "default datacenter" on
ESXi, however this can be done with [`vsphere_host`][data-source-vsphere-host]
as well.

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

~> **NOTE:** `name` is ignored on ESXi, and is not required.

## Attribute Reference

* `id` - The managed object ID of this datacenter.
* `datacenter_id` - (String) The managed object ID of this datacenter (same as
  `id`).
