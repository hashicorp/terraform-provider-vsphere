---
subcategory: "Storage"
layout: "vsphere"
page_title: "VMware vSphere: vm_storage_policy"
sidebar_current: "docs-vsphere-resource-vm-storage-policy"
description: |-
  Provides CRUD operations on vm storage policy profiles. These policies help create tag based rules for placement of a 
  VM on datastores. While placing a VM, compatible datastores can be filtered using these profiles.
---

# vsphere\_vm\_storage\_policy

The `vsphere_vm_storage_policy` resource can be used to create and manage storage 
policies. Using this storage policy, tag based placement rules can be created to 
place a VM on a particular tagged datastore.

## Example Usage

This example creates a storage policy with tag_rule having cat1 as tag_category and 
tag1, tag2 as the tags. While creating a VM, this policy can be referenced to place 
the VM in any of the compatible datastore tagged with these tags.


```hcl

data "vsphere_datacenter" "dc" {
  name = "DC"
}

data "vsphere_tag_category" "tag_category" {
  name = "cat1"
}

data "vsphere_tag" "tag1" {
  name        = "tag1"
  category_id = "${data.vsphere_tag_category.tag_category.id}"
}

data "vsphere_tag" "tag2" {
  name        = "tag2"
  category_id = "${data.vsphere_tag_category.tag_category.id}"
}

resource "vsphere_vm_storage_policy" "policy_tag_based_placement" {
  name        = "policy1"
  description = "description"

  tag_rules {
    tag_category                 = data.vsphere_tag_category.tag_category.name
    tags                         = [data.vsphere_tag.tag1.name, data.vsphere_tag.tag2.name]
    include_datastores_with_tags = true
  }

}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the storage policy.
* `description` - (Optional) Description of the storage policy. 
* `tag_rules` - (Required) List of tag rules. The tag category and tags to be associated to this storage policy.
  * `tag_category` - (Required) Name of the tag category.
  * `tags` - (Required) List of Name of tags to select from the given category.
  * `include_datastores_with_tags` - (Optional) Whether to include datastores with the given tags or exclude. Default 
     value is true i.e. include datastores with the given tags.

