---
subcategory: "Virtual Machine"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_content_library_item"
sidebar_current: "docs-vsphere-resource-content-library-item"
description: |-
  Creates an item in a Content Library. Each item can contain multiple files.
---

# vsphere\_content\_library_item

The `vsphere_content_library_item` resource can be used to create items in a Content Library.
`file_url` must be accessible from the vSphere environment as it will be downloaded from the
specified location and stored on the Content Library's storage backing.

## Example Usage

The example below creates an Ubuntu template on "Content Library Test".

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

resource "vsphere_content_library_item" "ubuntu1804" {
  name        = "Ubuntu Bionic 18.04"
  description = "Ubuntu template"
  library_id  = vsphere_content_library.library.id
  file_url = "https://fileserver/ubuntu/ubuntu-bionic-18.04-cloudimg.ovf
}

```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the item to be created in the Content Library.
* `library_id` - (Required) The ID of the Content Library the item should be created in.
* `file_url` - (Optional) File to import into the Content Library item. OVFs and
  OVAs will be parsed and associated files will also be imported.
* `source_uuid` - (Optional) Virtual machine UUID to clone to Content Library.
* `description` - (Optional) A description for the item.

## Attribute Reference

The only attribute this resource exports is the `id` of the resource, which is
a combination of the [managed object reference ID][docs-about-morefs] of the
cluster, and the name of the virtual machine group.

## Importing

An existing content library item can be [imported][docs-import] into this resource by
supplying the Content Library's ID. An example is below:

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_content_library_item  ubuntu1804 f42a4b25-844a-44ec-9063-a3a5e9cc88c7
```
