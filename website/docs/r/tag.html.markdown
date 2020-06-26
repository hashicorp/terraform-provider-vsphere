---
subcategory: "Inventory"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_tag"
sidebar_current: "docs-vsphere-resource-inventory-tag-resource"
description: |-
  Provides a vSphere tag resource. This can be used to manage tags in vSphere.
---

# vsphere\_tag

The `vsphere_tag` resource can be used to create and manage tags, which allow
you to attach metadata to objects in the vSphere inventory to make these
objects more sortable and searchable.

For more information about tags, click [here][ext-tags-general].

[ext-tags-general]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.vcenterhost.doc/GUID-E8E854DD-AA97-4E0C-8419-CE84F93C4058.html

~> **NOTE:** Tagging support is unsupported on direct ESXi connections and
requires vCenter 6.0 or higher.

## Example Usage

This example creates a tag named `terraform-test-tag`. This tag is assigned the
`terraform-test-category` category, which was created by the
[`vsphere_tag_category` resource][docs-tag-category-resource]. The resulting
tag can be assigned to VMs and datastores only, and can be the only value in
the category that can be assigned, as per the restrictions defined by the
category.

[docs-tag-category-resource]: /docs/providers/vsphere/r/tag_category.html

```hcl
resource "vsphere_tag_category" "category" {
  name        = "terraform-test-category"
  cardinality = "SINGLE"
  description = "Managed by Terraform"

  associable_types = [
    "VirtualMachine",
    "Datastore",
  ]
}

resource "vsphere_tag" "tag" {
  name        = "terraform-test-tag"
  category_id = "${vsphere_tag_category.category.id}"
  description = "Managed by Terraform"
}
```

## Using Tags in a Supported Resource

Tags can be applied to vSphere resources in Terraform via the `tags` argument
in any supported resource.

The following example builds on the above example by creating a
[`vsphere_virtual_machine`][docs-virtual-machine-resource] and applying the
created tag to it:

[docs-virtual-machine-resource]: /docs/providers/vsphere/r/virtual_machine.html

```hcl
resource "vsphere_tag_category" "category" {
  name        = "terraform-test-category"
  cardinality = "SINGLE"
  description = "Managed by Terraform"

  associable_types = [
    "VirtualMachine",
    "Datastore",
  ]
}

resource "vsphere_tag" "tag" {
  name        = "terraform-test-tag"
  category_id = "${vsphere_tag_category.category.id}"
  description = "Managed by Terraform"
}

resource "vsphere_virtual_machine" "web" {
  # ... other configuration ...

  tags = ["${vsphere_tag.tag.id}"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The display name of the tag. The name must be unique
  within its category.
* `category_id` - (Required) The unique identifier of the parent category in
  which this tag will be created. Forces a new resource if changed.
* `description` - (Optional) A description for the tag.

## Attribute Reference

The only attribute that is exported for this resource is the `id`, which is the
uniform resource name (URN) of this tag.

## Importing

An existing tag can be [imported][docs-import] into this resource by supplying
both the tag's category name and the name of the tag as a JSON string to
`terraform import`, as per the example below:

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_tag.tag \
  '{"category_name": "terraform-test-category", "tag_name": "terraform-test-tag"}'
```
