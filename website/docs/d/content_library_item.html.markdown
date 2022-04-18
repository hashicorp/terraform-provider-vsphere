---
subcategory: "Virtual Machine"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_content_library_item"
sidebar_current: "docs-vsphere-data-source-content-library-item"
description: |-
  Provides a VMware vSphere content library item data source.
---

# vsphere\_content\_library\_item

The `vsphere_content_library_item` data source can be used to discover the ID
of a content library item.

~> **NOTE:** This resource requires vCenter Server and is not available on
direct ESXi host connections.

## Example Usage

```hcl
data "vsphere_content_library" "library" {
  name = "Content Library"
}

data "vsphere_content_library_item" "item" {
  name       = "ovf-ubuntu-server-lts"
  type       = "ovf"
  library_id = data.vsphere_content_library.library.id
}

data "vsphere_content_library_item" "item" {
  name       = "tpl-ubuntu-server-lts"
  type       = "vm-template"
  library_id = data.vsphere_content_library.library.id
}

data "vsphere_content_library_item" "item" {
  name       = "iso-ubuntu-server-lts"
  type       = "iso"
  library_id = data.vsphere_content_library.library.id
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the content library item.
* `library_id` - (Required) The ID of the content library in which the item exists.
* `type` - (Required) The type for the content library item. One of `ovf`, `vm-template`, or `iso`

## Attribute Reference

* `id` - The UUID of the content library item.
