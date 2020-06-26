---
subcategory: "Virtual Machine"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_content_library"
sidebar_current: "docs-vsphere-resource-content-library"
description: |-
  Provides a VMware Content Library. Content libraries allow users to manage and share deployable content such as 
  virtual machines and vApps.
---

# vsphere\_content\_library

The `vsphere_content_library` resource can be used to manage Content Libraries.

~> **NOTE:** This resource requires vCenter and is not available on direct ESXi
connections.

## Example Usage

The example below creates a Content Library using `datastore1` as the storage backing.

[tf-vsphere-vm-resource]: /docs/providers/vsphere/r/virtual_machine.html

```hcl
data "vsphere_datacenter" "dc" {
  name = "dc1"
}

data "vsphere_datastore" "datastore" {
  name          = "datastore1"
  datacenter_id = data.vsphere_datacenter.dc.id
}

resource "vsphere_content_library" "library" {
  name            = "Content Library Test"
  storage_backing = data.vsphere_datastore.datastore.id
  description     = "A new source of content"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Content Library.
* `storage_backing` - (Required) The [managed object reference ID][docs-about-morefs] on which to store Content Library
  items.
* `description` - (Optional) A description of the Content Library.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider


## Attribute Reference

The only attribute this resource exports is the `id` of the resource, which is
a combination of the [managed object reference ID][docs-about-morefs] of the
cluster, and the name of the virtual machine group.

## Importing

An existing content library can be [imported][docs-import] into this resource by
supplying the Content Library's ID. An example is below:

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_content_library library f42a4b25-844a-44ec-9063-a3a5e9cc88c7
```
