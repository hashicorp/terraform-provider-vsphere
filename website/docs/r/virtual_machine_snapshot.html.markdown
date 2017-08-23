---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_virtual_machine_snapshot"
sidebar_current: "docs-vsphere-resource-virtual-machine-snapshot"
description: |-
  Provides a VMware vSphere virtual machine snapshot resource. This can be used to create, delete, and revert virtual machine's snapshot.
---

# vsphere\_virtual\_machine\_snapshot

Provides a VMware vSphere virtual machine snapshot resource. This can be used to create,
delete, and revert virtual machine's snapshot.

## Example Usage

```hcl
resource "vsphere_virtual_machine_snapshot" "demo1"{
 vm_id = "FolderName/VirtualMachine_Name"
 snapshot_name = "Snapshot Name"
 description = "This is Demo Snapshot"
 memory = "true"
 quiesce = "true"
 remove_children = "false" -  gettign used during delete vm
 consolidate = "true"
}

resource "vsphere_virtual_machine_snapshot_revert" "demo2"{
 vm_id = "FolderName/VirtualMachine_Name"
 snapshot_id = "snapshot-180"
}
```


## Argument Reference

The following arguments are supported:

1. for resource vsphere_virtual_machine_snapshot

* `vm_id` - (Required) The virtual machine id (FolderName/VM_Name)
* `snapshot_name` - (Required) New name for the snapshot.
* `description` - (Required) New description for the snapshot.
* `memory` - (Required) If the memory flag set to true, a dump of the internal state of the virtual machine is included in the snapshot.
* `quiesce` - (Required) If the quiesce flag set to true, and the virtual machine is powered on when the snapshot is taken, VMware Tools is used to quiesce the file system in the virtual machine.
* `datacenter` - (Optional) The name of a Datacenter in which the virtual machine exists for which we are taking snapshot
* `remove_children` - (Optional) Flag to specify removal of the entire snapshot subtree.
* `consolidate` - (optional) If set to true, the virtual disk associated with this snapshot will be merged with other disk if possible. Defaults to true.


2. for resource vsphere_virtual_machine_snapshot_revert

* `vm_id` - (Required) The virtual machine id (FolderName/VM_Name)
* `snapshot_id` - (Required) The snapshot id to which we need to revert back.
* `datacenter` - (Optional) The name of a Datacenter
* `suppress_power_on` - (Optional) If set to true, the virtual machine will not be powered on regardless of the power state when the snapshot was created. Default to false. 






