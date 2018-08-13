---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_virtual_disk"
sidebar_current: "docs-vsphere-resource-vm-virtual-disk"
description: |-
  Provides a VMware virtual disk resource.  This can be used to create and delete virtual disks.
---

# vsphere\_virtual\_disk

The `vsphere_virtual_disk` resource can be used to create virtual disks outside
of any given [`vsphere_virtual_machine`][docs-vsphere-virtual-machine]
resource. These disks can be attached to a virtual machine by creating a disk
block with the [`attach`][docs-vsphere-virtual-machine-disk-attach] parameter.

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
  be created.  This needs to end in `.vmdk`.
* `datastore` - (Required) The name of the datastore in which to create the
  disk.
* `size` - (Required) Size of the disk (in GB).
* `datacenter` - (Optional) The name of the datacenter in which to create the
  disk. Can be omitted when when ESXi or if there is only one datacenter in
  your infrastructure.
* `type` - (Optional) The type of disk to create. Can be one of
  `eagerZeroedThick`, `lazy`, or `thin`. Default: `eagerZeroedThick`. For
  information on what each kind of disk provisioning policy means, click
  [here][docs-vmware-vm-disk-provisioning].

[docs-vmware-vm-disk-provisioning]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.vm_admin.doc/GUID-4C0F4D73-82F2-4B81-8AA7-1DD752A8A5AC.html

* `adapter_type` - (Optional) The adapter type for this virtual disk. Can be
  one of `ide`, `lsiLogic`, or `busLogic`.  Default: `lsiLogic`.

~> **NOTE:** `adapter_type` is **deprecated**: it does not dictate the type of
controller that the virtual disk will be attached to on the virtual machine.
Please see the [`scsi_type`][docs-vsphere-virtual-machine-scsi-type] parameter
in the `vsphere_virtual_machine` resource for information on how to control
disk controller types. This parameter will be removed in future versions of the
vSphere provider.

[docs-vsphere-virtual-machine-scsi-type]: /docs/providers/vsphere/r/virtual_machine.html#scsi_type

* `create_directories` - (Optional) Tells the resource to create any
  directories that are a part of the `vmdk_path` parameter if they are missing.
  Default: `false`.

~> **NOTE:** Any directory created as part of the operation when
`create_directories` is enabled will not be deleted when the resource is
destroyed.
