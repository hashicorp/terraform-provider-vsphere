---
layout: "vsphere"
page_title: "VMware vSphere: Role"
sidebar_current: "docs-vsphere-data-source-vsphere-role"
description: |-
  A data source that can be used to fetch details of a vsphere role given its name/label.
---

# vsphere\_role

The `vsphere_role` data source can be used to discover the id and privileges associated
with a role given its name or display label in vsphere UI.


## Example Usage

```hcl
data "vsphere_role" "role1" {
  label = "Virtual machine user (sample)"
}
```

## Argument Reference

The following arguments are supported:

* `label` - (Required) The label of the role.

## Attribute Reference

* `id` - The id of the role.
* `description` - The description of the role.
* `role_privileges` - The privileges associated with the role.
* `label` - The display label of the role.

