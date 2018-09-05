---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_vapp_container"
sidebar_current: "docs-vsphere-data-source-resource-pool"
description: |-
  Provides a vSphere vApp container data source. This can be used to get the general attributes of a vSphere vApp container.
---

# vsphere\_resource\_pool

The `vsphere_vapp_container` data source can be used to discover the ID of a
vApp container in vSphere. This is useful to fetch the ID of a vApp container
that you want to use to create virtual machines in using the
[`vsphere_virtual_machine`][docs-virtual-machine-resource] resource. 

[docs-virtual-machine-resource]: /docs/providers/vsphere/r/virtual_machine.html

## Example Usage

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc1"
}

data "vsphere_vapp_container" "pool" {
  name          = "vapp-container-1"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the vApp container. This can be a name or
  path.
* `datacenter_id` - (Required) The [managed object reference
  ID][docs-about-morefs] of the datacenter the vApp container is located in.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

## Attribute Reference

Currently, the only exported attribute from this data source is `id`, which
represents the ID of the vApp container that was looked up.
