---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_entity_permission"
sidebar_current: "docs-vsphere-data-source-entity-permission"
description: |-
  Provides a vSphere entity permission data source. This can be used to reference entity permissions not managed in Terraform.
---

# vsphere\_entity\_permission

The `vsphere_entity_permission` data source can be used to reference entity permissions that are not managed by Terraform.

## Example Usage

```hcl
data "vsphere_entity_permission" "admin" {
  principal = "VSPHERE.LOCAL\\Administrator"
}
```

## Argument Reference

* `principal` - (Required) The name of the user/group.
* `folder_path` - (Optional) The folder path that the entity applied permissions to. Default to "/"

## Attribute Reference

* `role_id` - The role id that is associated to.
* `propagate` - Is propagation enabled
* `group` - Is the resulted entity permission a group
