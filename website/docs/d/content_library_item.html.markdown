---
subcategory: "Virtual Machine"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_content_library_item"
sidebar_current: "docs-vsphere-data-source-content-library-item"
description: |-
  Provides a VMware Content Library item data source.
---

# vsphere\_content\_library\_item

The `vsphere_content_library_item` data source can be used to discover the ID of a Content Library item.

~> **NOTE:** This resource requires vCenter and is not available on direct ESXi
connections.

## Example Usage

```hcl
data "vsphere_content_library" "library" {
  name = "Content Library Test"
}

data "vsphere_content_library_item" "item" {
  name       = "Ubuntu Bionic 18.04"
  library_id = data.vsphere_content_library.library.id
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Content Library.
* `library_id` - (Required) The ID of the Content Library the item exists in.


## Attribute Reference

* `id` - The UUID of the Content Library item.
* `type` - The Content Library type. Can be ovf, iso, or vm-template.
