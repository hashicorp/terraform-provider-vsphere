---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_distributed_virtual_switch"
sidebar_current: "docs-vsphere-data-source-dvs"
description: |-
  A data source that can be used to get the ID of a host.
---

# vsphere\_distributed\_virtual\_switch

The `vsphere_distributed_virtual_switch` data source can be used to discover the ID 
of a vSphere distributed virtual switch. This can then be used with resources or 
data sources that require a distributed virtual switch managed object reference ID.

## Example Usage

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc1"
}

data "vsphere_distributed_virtual_switch" "dvs" {
  name          = "myDVS"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (String) The name of the distributed virtual switch. This can be a name 
  or path.
* `datacenter_id` - (String, required) The managed object reference ID of a
  datacenter.

## Attribute Reference

The only exported attribute is `id`, which is the managed object ID of this
distributed virtual switch.
