---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_virtual_machine"
sidebar_current: "docs-vsphere-data-source-virtual-machine"
description: |-
  Provides a vSphere virtual machine data source. This can be used to get data from a virtual machine or template.
---

# vsphere\_virtual\_machine

The `vsphere_virtual_machine` data source can be used to find the UUID of an
existing virtual machine or template. Its most relevant purpose is for finding
the UUID of a template to be used as the source for cloning into a new
[`vsphere_virtual_machine`][docs-virtual-machine-resource] resource. It also
reads the guest ID so that can be supplied as well.

[docs-virtual-machine-resource]: /docs/providers/vsphere/r/virtual_machine.html

## Example Usage

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc1"
}

data "vsphere_virtual_machine" "template" {
  name          = "test-vm-template"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the virtual machine. This can be a name or
  path.
* `datacenter_id` - (Optional) The managed object reference ID of the
  datacenter the virtual machine is located in. This can be omitted if the
  search path used in `name` is an absolute path. For default datacenters, use
  the `id` attribute from an empty `vsphere_datacenter` data source.

## Attribute Reference

The following attributes are exported:

* `id`: The UUID of the virtual machine or template.
* `guest_id`: The guest ID of the virtual machine or template.
* `alternate_guest_name`: The alternate guest name of the virtual machine when
  guest_id is a non-specific operating system, like `otherGuest`.
* `scsi_type`: The common type of all SCSI controllers on this virtual machine.
  Will be one of `lsilogic` (LSI Logic Parallel), `lsilogic-sas` (LSI Logic
  SAS), `pvscsi` (VMware Paravirtual), `buslogic` (BusLogic), or `mixed` when
  there are multiple controller types.
* `disk_sizes`: The sizes, in GiB, of each of the virtual disks on this virtual
  machine or template. These are sorted by bus and unit number so that they can
  be applied to a `vsphere_virtual_machine` resource in the order the resource
  expects while cloning. This is useful for discovering the required size of a
  disk while performing a linked clone, as the disk size of a linked clone must
  be the same as its source.
* `network_interface_types`: The network interface types for each network
  interface found on the virtual machine, in device bus order. Will be one of
  `e1000`, `e1000e`, `pcnet32`, `sriov`, `vmxnet2`, or `vmxnet3`.

~> **NOTE:** Keep in mind when using the results of `scsi_type` and
`network_interface_types`, that the `vsphere_virtual_machine` resource only
supports a subset of the types returned from this data source. See the
[resource docs][docs-virtual-machine-resource] for more details.
