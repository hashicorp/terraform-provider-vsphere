---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_distributed_virtual_switch"
sidebar_current: "docs-vsphere-data-source-distributed-virtual-switch"
description: |-
  Provides a vSphere distributed virtual switch data source. This can be used to get select attributes of a DVS.
---

# vsphere\_distributed\_virtual\_switch

The `vsphere_distributed_virtual_switch` data source can be used to discover
the ID and uplink data of a of a vSphere distributed virtual switch (DVS). This
can then be used with resources or data sources that require a DVS, such as the
[`vsphere_distributed_port_group`][distributed-port-group] resource, for which
an example is shown below.

[distributed-port-group]: /docs/providers/vsphere/r/distributed_port_group.html

~> **NOTE:** This data source requires vCenter and is not available on direct
ESXi connections.

## Example Usage

The following example locates a DVS that is named `terraform-test-dvs`, in the
datacenter `dc1`. It then uses this DVS to set up a
`vsphere_distributed_port_group` resource that uses the first uplink as a
primary uplink and the second uplink as a secondary.

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc1"
}

data "vsphere_distributed_virtual_switch" "dvs" {
  name          = "terraform-test-dvs"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_distributed_port_group" "pg" {
  name                            = "terraform-test-pg"
  distributed_virtual_switch_uuid = "${data.vsphere_distributed_virtual_switch.dvs.id}"

  active_uplinks  = ["${data.vsphere_distributed_virtual_switch.dvs.uplinks[0]}"]
  standby_uplinks = ["${data.vsphere_distributed_virtual_switch.dvs.uplinks[1]}"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the distributed virtual switch. This can be a
  name or path.
* `datacenter_id` - (Optional) The [managed object reference
  ID][docs-about-morefs] of the datacenter the DVS is located in. This can be
  omitted if the search path used in `name` is an absolute path. For default
  datacenters, use the id attribute from an empty `vsphere_datacenter` data
  source.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

## Attribute Reference

The following attributes are exported:

* `id`: The UUID of the distributed virtual switch.
* `uplinks`: The list of the uplinks on this DVS, as per the
  [`uplinks`][distributed-virtual-switch-uplinks] argument to the
  [`vsphere_distributed_virtual_switch`][distributed-virtual-switch-resource]
  resource.

[distributed-virtual-switch-uplinks]: /docs/providers/vsphere/r/distributed_virtual_switch.html#uplinks
[distributed-virtual-switch-resource]: /docs/providers/vsphere/r/distributed_virtual_switch.html
