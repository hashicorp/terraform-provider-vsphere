---
subcategory: "Virtual Machine"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_guest_os_customization"
sidebar_current: "docs-vsphere-data-guest-os-customization"
description: |-
  Provides a VMware vSphere guest customization spec data source. This can be used to apply the customization spec when virtual machine is cloned
---

# vsphere\_guest\_os\_customization

The `vsphere_guest_os_customization` data source can be used to discover the details about a customization specification for a guest operating system.

Suggested change
~> **NOTE:** The name attribute is the unique identifier for the customization specification per vCenter Server instance.


## Example Usage

```hcl
  data "vsphere_guest_os_customization" "gosc1" {
    name          = "linux-spec"
  }
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the customization specification is the unique identifier per vCenter Server instance.
## Attribute Reference

* `type` - The type of customization specification: One among: Windows, Linux.
* `description` - The description for the customization specification.
* `last_update_time` - The time of last modification to the customization specification.
* `change_version` - The number of last changed version to the customization specification.
* `spec` - Container object for the guest operating system properties to be customized. See [virtual machine customizations](virtual_machine#virtual-machine-customizations)
