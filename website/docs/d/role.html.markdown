---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_role"
sidebar_current: "docs-vsphere-data-source-role"
description: |-
  Provides a vSphere role data source. This can be used to reference roles not managed in Terraform.
---

# vsphere\_role

The `vsphere_role` data source can be used to reference roles that are not managed by Terraform.
it provide both name search and role_id search for roles

## Example Usage

```hcl
# Using Name
data "vsphere_role" "name" {
  name = "Admin"
}

# Using Role ID
data "vsphere_role" "role-id" {
  role_id = -1
}
```

## Argument Reference

* `name` - (Required) The name of the role. `role_id` can't be use if using this
* `role_id` - (Required) The internal id of the role. `name` can't be use if using this

## Attribute Reference

* `name` - The name of the role.
* `role_id` - The internal id of the role.
* `permissions` - List of associated permissions to the role. [Permissions Reference](https://github.com/terraform/terraform-provider-vsphere/blob/f-role-permissions/vsphere/internal/helper/role/role_helper.go#L13)
