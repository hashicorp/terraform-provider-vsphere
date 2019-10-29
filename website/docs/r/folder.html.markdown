---
subcategory: "Inventory"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_folder"
sidebar_current: "docs-vsphere-resource-inventory-folder"
description: |-
  Provides a VMware vSphere folder resource. This can be used to manage vSphere inventory folders.
---

# vsphere\_folder

The `vsphere_folder` resource can be used to manage vSphere inventory folders.
The resource supports creating folders of the 5 major types - datacenter
folders, host and cluster folders, virtual machine folders, datastore folders,
and network folders.

Paths are always relative to the specific type of folder you are creating.
Subfolders are discovered by parsing the relative path specified in `path`, so
`foo/bar` will create a folder named `bar` in the parent folder `foo`, as long
as that folder exists.

## Example Usage

The basic example below creates a virtual machine folder named
`terraform-test-folder` in the default datacenter's VM hierarchy. 

```hcl
data "vsphere_datacenter" "dc" {}

resource "vsphere_folder" "folder" {
  path          = "terraform-test-folder"
  type          = "vm"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}
```

### Example with subfolders

The below example builds off of the above by first creating a folder named
`terraform-test-parent`, and then locating `terraform-test-folder` in that
folder. To ensure the parent is created first, we create an interpolation
dependency off the parent's `path` attribute.

Note that if you change parents (for example, went from the above basic
configuration to this one), your folder will be moved to be under the correct
parent.

```hcl
data "vsphere_datacenter" "dc" {}

resource "vsphere_folder" "parent" {
  path          = "terraform-test-parent"
  type          = "vm"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_folder" "folder" {
  path          = "${vsphere_folder.parent.path}/terraform-test-folder"
  type          = "vm"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}
```

## Argument Reference

The following arguments are supported:

* `path` - (Required) The path of the folder to be created. This is relative to
  the root of the type of folder you are creating, and the supplied datacenter.
  For example, given a default datacenter of `default-dc`, a folder of type
  `vm` (denoting a virtual machine folder), and a supplied folder of
  `terraform-test-folder`, the resulting path would be
  `/default-dc/vm/terraform-test-folder`.

~> **NOTE:** `path` can be modified - the resulting behavior is dependent on
what section of `path` you are modifying. If you are modifying the parent (so
any part before the last `/`), your folder will be moved to that new parent. If
modifying the name (the part after the last `/`), your folder will be renamed.

* `type` - (Required) The type of folder to create. Allowed options are
  `datacenter` for datacenter folders, `host` for host and cluster folders,
  `vm` for virtual machine folders, `datastore` for datastore folders, and
  `network` for network folders. Forces a new resource if changed.
* `datacenter_id` - The ID of the datacenter the folder will be created in.
  Required for all folder types except for datacenter folders. Forces a new
  resource if changed.
* `tags` - (Optional) The IDs of any tags to attach to this resource. See
  [here][docs-applying-tags] for a reference on how to apply tags.

[docs-applying-tags]: /docs/providers/vsphere/r/tag.html#using-tags-in-a-supported-resource

~> **NOTE:** Tagging support is unsupported on direct ESXi connections and
requires vCenter 6.0 or higher.

* `custom_attributes` - (Optional) Map of custom attribute ids to attribute 
  value strings to set for folder. See [here][docs-setting-custom-attributes] 
  for a reference on how to set values for custom attributes.

[docs-setting-custom-attributes]: /docs/providers/vsphere/r/custom_attribute.html#using-custom-attributes-in-a-supported-resource

~> **NOTE:** Custom attributes are unsupported on direct ESXi connections 
and require vCenter.

## Attribute Reference

The only attribute that this resource exports is the `id`, which is set to the
[managed object ID][docs-about-morefs] of the folder.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

## Importing

An existing folder can be [imported][docs-import] into this resource via
its full path, via the following command:

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_folder.folder /default-dc/vm/terraform-test-folder
```

The above command would import the folder from our examples above, the VM
folder named `terraform-test-folder` located in the datacenter named
`default-dc`.
