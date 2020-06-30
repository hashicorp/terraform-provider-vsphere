---
subcategory: "Virtual Machine"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_content_library"
sidebar_current: "docs-vsphere-data-source-content-library"
description: |-
  Provides a VMware Content Library data source.
---

# vsphere\_content\_library

The `vsphere_content_library` data source can be used to discover the ID of a Content Library.

~> **NOTE:** This resource requires vCenter and is not available on direct ESXi
connections.

## Example Usage

```hcl
data "vsphere_content_library" "library" {
  name = "Content Library Test"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Content Library.

## Attribute Reference

The only attribute this resource exports is the `id` of the resource, which is
a combination of the [managed object reference ID][docs-about-morefs] of the
cluster, and the name of the virtual machine group.
