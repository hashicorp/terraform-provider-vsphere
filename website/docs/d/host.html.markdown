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

* `datacenter_id` - (Required) The managed object reference ID of a datacenter.
* `name` - (Optional) The name of the host. This can be a name or path.	Can be
  omitted if there is only one host in your inventory.

~> **NOTE:** When used against an ESXi host directly, this data source _always_
fetches the server's host object ID, regardless of what is entered into `name`.

## Attribute Reference

The only exported attribute is `id`, which is the managed object ID of this
host.
