---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_role"
sidebar_current: "docs-vsphere-resource-role"
description: |-
  Provides a vSphere role resource. This can be used to manage roles in vSphere.
---

# vsphere\_role

The `vsphere_role` resource can be used to create and manage roles,
Which can be use for specifying a set of permissions to a particular principal

## Example Usage

```hcl
resource "vsphere_role" "default" {
	name = "TestRole"

	permissions = [
		"VirtualMachine.State.CreateSnapshot",
	]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the role.
* `permissions` - (Required) List of permissions that is associate with the role. [Permissions Reference](https://github.com/terraform/terraform-provider-vsphere/blob/f-role-permissions/vsphere/internal/helper/role/role_helper.go#L13)
