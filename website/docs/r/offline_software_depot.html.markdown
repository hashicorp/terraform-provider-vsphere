---
subcategory: "Lifecycle"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_offline_software_depot"
sidebar_current: "docs-vsphere-resource-offline-software-depot"
description: |-
  Provides a VMware vSphere offline software depot resource..
---

# vsphere\_offline\_software\_depot

Provides a VMware vSphere offline software depot resource.

## Example Usages

**Create an offline depot**

```hcl
data "vsphere_offline_software_depot" "depot" {
  location = "https://your.company.server/path/to/your/files"
}
```

## Argument Reference

* `location` - The URL where the depot source is hosted.

## Attribute Reference

* `component` - The list of custom components in the depot.
  * `key` - The identifier of the component.
  * `version` - The list of available versions of the component.
  * `display_name` - The name of the component. Useful for easier identification.
