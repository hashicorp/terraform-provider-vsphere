---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_virtual_disk"
sidebar_current: "docs-vsphere-resource-vm-virtual-disk"
description: |-
  Provides a VMware virtual disk resource.  This can be used to create and delete virtual disks.
---

# vsphere\_virtual\_disk

The `vsphere_virtual_disk` resource can be used to create virtual disks
external to any [`vsphere_virtual_machine`][docs-vsphere-virtual-machine]
resource. These disks can be attached to a virtual machine by creating a disk
sub-resource with the [`attach`][docs-vsphere-virtual-machine-disk-attach]
parameter.

[docs-vsphere-virtual-machine]: /docs/providers/vsphere/r/virtual_machine.html
[docs-vsphere-virtual-machine-disk-attach]: /docs/providers/vsphere/r/virtual_machine.html#attach

## Example Usage

```hcl
resource "vsphere_virtual_disk" "myDisk" {
  size         = 2
  vmdk_path    = "myDisk.vmdk"
  datacenter   = "Datacenter"
  datastore    = "local"
  type         = "thin"
}
```

## Argument Reference

The following arguments are supported:

~> **NOTE:** All fields in the `vsphere_virtual_disk` resource are currently
immutable and force a new resource if changed.

* `vmdk_path` - (Required) The path, including filename, of the virtual disk to
  be created.  This should end with `.vmdk`.
* `datastore` - (Required) The name of the datastore in which to create the
  disk.
* `size` - (Required) Size of the disk (in GB).
* `datacenter` - (Optional) The name of the datacenter in which to create the
  disk. Can be omitted when when ESXi or if there is only one datacenter in
  your infrastructure.
* `type` - (Optional) The type of disk to create. Can be one of
  `eagerZeroedThick`, `lazy`, or `thin`. Default: `eagerZeroedThick`.
* `adapter_type` - (Optional) The adapter type for this virtual disk. Can be
  one of `ide`, `lsiLogic`, or `busLogic`.  Default: `lsiLogic`.

~> **NOTE:** `adapter_type` is **deprecated**: it does not dictate the type of
controller that the virtual disk will be attached to on the virtual machine.
Please see the [`scsi_type`][docs-vsphere-virtual-machine-scsi-type] parameter
in the `vsphere_virtual_machine` resource for information on how to control
disk controller types. This parameter will be removed in future versions of the
vSphere provider.

[docs-vsphere-virtual-machine-scsi-type]: /docs/providers/vsphere/r/virtual_machine.html#scsi_type
