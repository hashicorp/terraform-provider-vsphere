---
subcategory: "Security"
layout: "vsphere"
page_title: "VMware vSphere: Role"
sidebar_current: "docs-vsphere-data-source-vsphere-role"
description: |-
  Provides a vSphere role data source.
---

# vsphere\_role

The `vsphere_role` data source can be used to discover the `id` and privileges associated
with a role given its name or display label.


## Example Usage

```hcl
data "vsphere_role" "terraform_role" {
  label = "Terraform to vSphere Integration Role"
}
```

## Argument Reference

The following arguments are supported:

* `label` - (Required) The label of the role.

## Attribute Reference

* `id` - The ID of the role.
* `description` - The description of the role.
* `role_privileges` - The privileges associated with the role.
* `label` - The display label of the role.
