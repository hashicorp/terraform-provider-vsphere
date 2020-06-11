---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_dynamic"
sidebar_current: "docs-vsphere-data-source-dynamic"
description: |-
  A data source that can be used to get the [managed object reference ID][docs-about-morefs]
  of any tagged managed object in vCenter.
---
[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

# vsphere_dynamic

The `vsphere_dynamic` data source can be used to get the [managed object 
  reference ID][docs-about-morefs] of any tagged managed object in vCenter
  by providing a list of tag IDs and an optional regular expression to filter
  objects by name.
   
## Example Usage

```hcl
data "vsphere_tag_cetegory" "cat" {
  name = "SomeCategory"
}

data "vsphere_tag" "tag1" {
  name        = "FirstTag"
  category_id = data.vsphere_tag_category.cat.id
}

data "vsphere_tag" "tag2" {
  name        = "SecondTag"
  category_id = data.vsphere_tag_category.cat.id
}

data "vsphere_dynamic" "dyn" {
  filter     = [data.vsphere_tag.tag1.id, data.vsphere_tag.tag1.id]
  name_regex = "ubuntu"
  type       = "Datacenter"
}
```
## Argument Reference

The following arguments are supported:

* `filter` - (Required) A list of tag IDs that must be present on an object to
  be a match. 
* `name_regex` - (Optional) A regular expression that will be used to match
  the object's name.
* `type` - (Optional) The managed object type the returned object must match.
  For a full list, click [here](https://code.vmware.com/apis/196/vsphere).

## Attribute Reference

* `id` - The device ID of the matched managed object.