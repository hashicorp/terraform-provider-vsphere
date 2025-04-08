---
subcategory: "Virtual Machine"
page_title: "VMware vSphere: vsphere_content_library"
sidebar_current: "docs-vsphere-data-source-content-library"
description: |-
  Provides a VMware vSphere content library data source.
---

# vsphere_content_library

The `vsphere_content_library` data source can be used to discover the ID of a
content library.

~> **NOTE:** This resource requires vCenter and is not available on direct ESXi
host connections.

## Example Usage

```hcl
data "vsphere_content_library" "content_library" {
  name = "Content Library"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the content library.

## Attribute Reference

The only attribute this resource exports is the `id` of the resource, which is
a combination of the [managed object reference ID][docs-about-morefs] of the
cluster, and the name of the virtual machine group.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider
