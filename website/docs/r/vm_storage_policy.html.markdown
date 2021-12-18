---
subcategory: "Storage"
layout: "vsphere"
page_title: "VMware vSphere: vm_storage_policy"
sidebar_current: "docs-vsphere-resource-vm-storage-policy"
description: |-
  Storage policies can select the most appropriate datastore for the virtual machine and enforce the required level of service. 
---

# vsphere\_vm\_storage\_policy

The `vsphere_vm_storage_policy` resource can be used to create and manage storage 
policies. Using this resource, tag based placement rules can be created to 
place virtual machines on a datastore with matching tags. If storage requirements for the applications on the virtual machine change, you can modify the storage policy that was originally applied to the virtual machine.

## Example Usage

The following example creates storage policies with `tag_rules` base on sets of environment, service level, and replication attributes.

In this example, tags are first applied to datastores.

```hcl
data "vsphere_tag_category" "environment" {
  name = "environment"
}

data "vsphere_tag_category" "service_level" {
  name = "service_level"
}

data "vsphere_tag_category" "replication" {
  name = "replication"
}

data "vsphere_tag" "production" {
  name        = "production"
  category_id = "data.vsphere_tag_category.environment.id"
}

data "vsphere_tag" "development" {
  name        = "development"
  category_id = "data.vsphere_tag_category.environment.id"
}

data "vsphere_tag" "platinum" {
  name        = "platinum"
  category_id = "data.vsphere_tag_category.service_level.id"
}

data "vsphere_tag" "gold" {
  name        = "platinum"
  category_id = "data.vsphere_tag_category.service_level.id"
}

data "vsphere_tag" "silver" {
  name        = "silver"
  category_id = "data.vsphere_tag_category.service_level.id"
}

data "vsphere_tag" "bronze" {
  name        = "bronze"
  category_id = "data.vsphere_tag_category.service_level.id"
}

data "vsphere_tag" "replicated" {
  name        = "replicated"
  category_id = "data.vsphere_tag_category.replication.id"
}

data "vsphere_tag" "non_replicated" {
  name        = "non_replicated"
  category_id = "data.vsphere_tag_category.replication.id"
}

resource "vsphere_vmfs_datastore" "prod_datastore" {
  # ... other configuration ...
  tags = [
    "data.vsphere_tag.production.id",
    "data.vsphere_tag.platinum.id",
    "data.vsphere_tag.replicated.id"
  ]
  # ... other configuration ...
}

resource "vsphere_nas_datastore" "dev_datastore" {
  # ... other configuration ...
  tags = [
    "data.vsphere_tag.development.id",
    "data.vsphere_tag.silver.id",
    "data.vsphere_tag.non_replicated.id"
  ]
  # ... other configuration ...
}
```

Next, storage policies are created and `tag_rules` are applied.

```hcl
resource "vsphere_vm_storage_policy" "prod_platinum_replicated" {
  name        = "prod_platinum_replicated"
  description = "prod_platinum_replicated"

  tag_rules {
    tag_category                 = data.vsphere_tag_category.environment.name
    tags                         = [data.vsphere_tag.production.name]
    include_datastores_with_tags = true
  }
  tag_rules {
    tag_category                 = data.vsphere_tag_category.service_level.name
    tags                         = [data.vsphere_tag.platinum.name]
    include_datastores_with_tags = true
  }
  tag_rules {
    tag_category                 = data.vsphere_tag_category.replication.name
    tags                         = [data.vsphere_tag.replicated.name]
    include_datastores_with_tags = true
  }
}

resource "vsphere_vm_storage_policy" "dev_silver_nonreplicated" {
  name        = "dev_silver_nonreplicated"
  description = "dev_silver_nonreplicated"

  tag_rules {
    tag_category                 = data.vsphere_tag_category.environment.name
    tags                         = [data.vsphere_tag.development.name]
    include_datastores_with_tags = true
  }
  tag_rules {
    tag_category                 = data.vsphere_tag_category.service_level.name
    tags                         = [data.vsphere_tag.silver.name]
    include_datastores_with_tags = true
  }
  tag_rules {
    tag_category                 = data.vsphere_tag_category.replication.name
    tags                         = [data.vsphere_tag.non_replicated.name]
    include_datastores_with_tags = true
  }
}
```

Lasttly, when creating a virtual machine resource, a storage policy can be specificed to direct virtual machine placement to a datastore which matches the policy's `tags_rules`.

```hcl
data "vsphere_storage_policy" "prod_platinum_replicated" {
  name = "prod_platinum_replicated"
}

data "vsphere_storage_policy" "dev_silver_nonreplicated" {
  name = "dev_silver_nonreplicated"
}

resource "vsphere_virtual_machine" "prod_vm" {
  # ... other configuration ...
  storage_policy_id = data.vsphere_storage_policy.storage_policy.prod_platinum_replicated.id
  # ... other configuration ...
}

resource "vsphere_virtual_machine" "dev_vm" {
  # ... other configuration ...
  storage_policy_id = data.vsphere_storage_policy.storage_policy.dev_silver_nonreplicated.id
  # ... other configuration ...
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the storage policy.
* `description` - (Optional) Description of the storage policy. 
* `tag_rules` - (Required) List of tag rules. The tag category and tags to be associated to this storage policy.
  * `tag_category` - (Required) Name of the tag category.
  * `tags` - (Required) List of Name of tags to select from the given category.
  * `include_datastores_with_tags` - (Optional) Include datastores with the given tags or exclude. Default `true`.