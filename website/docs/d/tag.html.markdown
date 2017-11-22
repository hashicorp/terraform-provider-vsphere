---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_tag"
sidebar_current: "docs-vsphere-data-source-tag-data-source"
description: |-
  Provides a vSphere tag data source. This can be used to reference tags not managed in Terraform.
---

# vsphere\_tag

The `vsphere_tag` data source can be used to reference tags that are not
managed by Terraform. Its attributes are exactly the same as the [`vsphere_tag`
resource][resource-tag], and, like importing, the data source takes a name and
category to search on. The `id` and other attributes are then populated with
the data found by the search.

[resource-tag]: /docs/providers/vsphere/r/tag.html

~> **NOTE:** Tagging support is unsupported on direct ESXi connections and
requires vCenter 6.0 or higher.

## Example Usage

```hcl
data "vsphere_tag_category" "category" {
  name = "terraform-test-category"
}

data "vsphere_tag" "tag" {
  name        = "terraform-test-tag"
  category_id = "${data.vsphere_tag_category.category.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the tag.
* `category_id` - (Required) The ID of the tag category the tag is located in.

## Attribute Reference

In addition to the `id` being exported, all of the fields that are available in
the [`vsphere_tag` resource][resource-tag] are also populated. See that page
for further details.
