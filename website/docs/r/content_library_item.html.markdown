---
subcategory: "Virtual Machine"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_content_library_item"
sidebar_current: "docs-vsphere-resource-content-library-item"
description: |-
  Creates an item in a vSphere content library. Each item can contain multiple files.
---

# vsphere\_content\_library_item

The `vsphere_content_library_item` resource can be used to create items in a
vSphere content library. The `file_url` must be accessible from the vSphere
environment as it will be downloaded from the specified location and stored
on the content library's storage backing.

## Example Usage

The first example below imports an OVF Template to a content
library.

[tf-vsphere-vm-resource]: /docs/providers/vsphere/r/virtual_machine.html

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc-01"
}

data "vsphere_content_library" "content_library" {
  name = "clb-01"
}

resource "vsphere_content_library_item" "content_library_item" {
  name        = "ovf-linux-ubuntu-server-lts"
  description = "Ubuntu Server LTS OVF Template"
  file_url    = "https://releases.example.com/ubuntu/ubuntu/ubuntu-live-server-amd64.ovf"
  library_id  = data.vsphere_content_library.content_library.id
}
```

The next example imports an .iso image to a content library.

[tf-vsphere-vm-resource]: /docs/providers/vsphere/r/virtual_machine.html

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc-01"
}

data "vsphere_content_library" "content_library" {
  name = "clb-01"
}

resource "vsphere_content_library_item" "content_library_item" {
  name        = "iso-linux-ubuntu-server-lts"
  description = "Ubuntu Server LTS .iso"
  type        = "iso"
  file_url    = "https://releases.example.com/ubuntu/ubuntu-live-server-amd64.iso"
  library_id  = data.vsphere_content_library.content_library.id
}
```

The last example imports a virtual machine image to a content library from an
existing virtual machine.

[tf-vsphere-vm-resource]: /docs/providers/vsphere/r/virtual_machine.html

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc-01"
}

data "vsphere_content_library" "content_library" {
  name = "clb-01"
}

resource "vsphere_content_library_item" "content_library_item" {
  name        = "tpl-linux-ubuntu-server-lts"
  description = "Ubuntu Server LTS"
  source_uuid = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  library_id  = data.vsphere_content_library.content_library.id
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the item to be created in the content library.
* `library_id` - (Required) The ID of the content library in which to create the item.
* `file_url` - (Optional) File to import as the content library item.
* `source_uuid` - (Optional) Virtual machine UUID to clone to content library.
* `description` - (Optional) A description for the content library item.
* `type` - (Optional) Type of content library item.
   One of "ovf", "iso", or "vm-template". Default: `ovf`.

## Attribute Reference

The only attribute this resource exports is the `id` of the resource, which is
a combination of the [managed object reference ID][docs-about-morefs] of the
cluster, and the name of the virtual machine group.

## Importing

An existing content library item can be [imported][docs-import] into this resource by
supplying the content library ID. An example is below:

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_content_library_item iso-linux-ubuntu-server-lts xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```
