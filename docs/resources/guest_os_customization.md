---
subcategory: "Virtual Machine"
page_title: "VMware vSphere: vsphere_guest_os_customization"
sidebar_current: "docs-vsphere-data-guest-os-customization"
description: |-
  Provides a VMware vSphere customization specification resource. This can be used to apply a customization specification to the guest operating system of a virtual machine after cloning.
---

# vsphere_guest_os_customization

The `vsphere_guest_os_customization` resource can be used to a customization specification for a guest operating system.

~> **NOTE:** The name attribute is unique identifier for the guest OS spec per VC.

## Example Usage

```hcl
resource "vsphere_guest_os_customization" "windows" {
  name = "windows"
  type = "Windows"
  spec {
    windows_options {
      run_once_command_list = ["command-1", "command-2"]
      computer_name         = "windows"
      auto_logon            = false
      auto_logon_count      = 0
      admin_password        = "VMware1!"
      time_zone             = 004
      workgroup             = "workgroup"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the customization specification is the unique identifier per vCenter Server instance.
* `type` - (Required) The type of customization specification: One among: Windows, Linux.
* `description` - (Optional) The description for the customization specification.
* `spec` - Container object for the Guest OS properties about to be customized . See [virtual machine customizations](virtual_machine#virtual-machine-customizations)

## Attribute Reference

* `last_update_time` - The time of last modification to the customization specification.
* `change_version` - The number of last changed version to the customization specification.
