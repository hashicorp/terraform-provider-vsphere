---
subcategory: "Security"
layout: "vsphere"
page_title: "VMware vSphere: Role"
sidebar_current: "docs-vsphere-resource-role"
description: |-
  Provides CRUD operations on a vsphere role. A role can be created and privileges can be associated with it,
---

# vsphere\_role

The `vsphere_role` resource can be used to create and manage roles. Using this resource, privileges can be 
associated with the roles. The role can be used while granting permissions to an entity.

## Example Usage

This example creates a role with name my_terraform_role and privileges create, acknowledge for Alarm and 
create, move for Datacenter. While providing `role_privileges`, the id of the privilege has to be provided.
The format of the privilege id is privilege name preceded by its categories joined by a `.`.
For example a privilege with path `category->subcategory->privilege` should be provided as 
`category.subcategory.privilege`. Keep the `role_privileges` sorted alphabetically for a better user experience.

~> **NOTE:** While providing `role_privileges`, the id of the privilege and its categories are to be provided
joined by a `.` .

```hcl

resource vsphere_role "role1" {
  name = "my_terraform_role"
  role_privileges = ["Alarm.Acknowledge", "Alarm.Create", "Datacenter.Create", "Datacenter.Move"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the role.
* `role_privileges` - (Optional) The privileges to be associated with this role.

## Importing

An existing role can be imported into this resource by supplying the role id. An example is below:

```hcl
terraform import vsphere_role.role1 -709298051
```
~> **NOTE:** System roles can't be imported because they can't be modified or deleted.
Use [`vsphere_role` data source][ref-vsphere-role-data-source]
to read information about system roles.

[ref-vsphere-role-data-source]: /docs/providers/vsphere/d/vsphere_role.html
