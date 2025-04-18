---
subcategory: "Inventory"
page_title: "VMware vSphere: vsphere_folder"
sidebar_current: "docs-vsphere-data-source-inventory-folder"
description: |-
  Provides a VMware vSphere folder data source. This can be used to get the
  general attributes of a vSphere inventory folder.
---

# vsphere_folder

The `vsphere_folder` data source can be used to get the general attributes of a
vSphere inventory folder. The data source supports creating folders of the 5
major types - datacenter folders, host and cluster folders, virtual machine
folders, storage folders, and network folders.

Paths are absolute and must include the datacenter.

## Example Usage

```hcl
data "vsphere_folder" "folder" {
  path = "/dc-01/datastore-01/folder-01"
}
```

## Argument Reference

The following arguments are supported:

* `path` - (Required) The absolute path of the folder. For example, given a
  default datacenter of `default-dc`, a folder of type `vm`, and a folder name
  of `terraform-test-folder`, the resulting path would be
  `/default-dc/vm/terraform-test-folder`. The valid folder types to be used in
  the path are: `vm`, `host`, `datacenter`, `datastore`, or `network`.

## Attribute Reference

The only attribute that this resource exports is the `id`, which is set to the
[managed object ID][docs-about-morefs] of the folder.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider
