---
subcategory: "Virtual Machine"
page_title: "VMware vSphere: vsphere_guest_os_customization"
sidebar_current: "docs-vsphere-data-guest-os-customization"
description: |-
  Provides a VMware vSphere guest customization spec data source.
  This can be used to apply the customization spec when virtual machine is
  cloned.
---

# vsphere_guest_os_customization

The `vsphere_guest_os_customization` data source can be used to discover the
details about a customization specification for a guest operating system.

## Example Usage

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc-01"
}

data "vsphere_virtual_machine" "template" {
  name          = "windows-template"
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

data "vsphere_guest_os_customization" "windows" {
  name = "windows"
}

resource "vsphere_virtual_machine" "vm" {
  # ... other configuration ...
  template_uuid = data.vsphere_virtual_machine.template.id
  customization_spec {
    id = data.vsphere_guest_os_customization.windows.id
  }
  # ... other configuration ...
}
```

## Argument Reference

The following arguments are supported:

- `name` - (Required) The name of the customization specification is the unique
  identifier per vCenter Server instance. 

## Attribute Reference

- `type` - The type of customization specification: One among: Windows, Linux.
- `description` - The description for the customization specification.
- `last_update_time` - The time of last modification to the customization
  specification.
- `change_version` - The number of last changed version to the customization
  specification.
- `spec` - Container object for the guest operating system properties to be
  customized. See
  [virtual machine customizations](virtual_machine#virtual-machine-customizations)
