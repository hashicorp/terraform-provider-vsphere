---
subcategory: "Workload Management"
page_title: "VMware vSphere: vsphere_virtual_machine_class"
sidebar_current: "docs-vsphere-resource-virtual-machine-class"
description: |-
  Provides a VMware vSphere virtual machine class resource..
---

# vsphere_virtual_machine_class

Provides a resource for configuring a Virtual Machine class.

## Example Usages

### Create a Basic Class

```hcl
resource "vsphere_virtual_machine_class" "basic_class" {
  name   = "basic-class"
  cpus   = 4
  memory = 4096
}
```

### Create a Class with a vGPU

```hcl
resource "vsphere_virtual_machine_class" "vgp_class" {
  name               = "vgpu-class"
  cpus               = 4
  memory             = 4096
  memory_reservation = 100
  vgpu_devices       = ["vgpu1"]
}
```

## Argument Reference

* `name` - The name for the class.
* `cpus` - The number of CPUs.
* `memory` - The amount of memory in MB.
* `memory_reservation` - The percentage of memory reservation.
* `vgpu_devices` - The identifiers of the vGPU devices for the class. If this is set memory reservation needs to be 100.
