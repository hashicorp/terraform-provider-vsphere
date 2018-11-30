---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_entity_permission"
sidebar_current: "docs-vsphere-resource-entity-permission"
description: |-
  Provides a vSphere entity permission resource. This can be used to manage entity permissions in vSphere.
---

# vsphere\_entity\_permission

The `vsphere_entity_permission` resource can be used to create and manage entity permissions,
which allow association between a principal(user or group) to role with a given folder path.

## Example Usage

```hcl
data "vsphere_role" "default" {
	name = "Admin"
}

resource "vsphere_entity_permission" "default" {
	principal = "VSPHERE.LOCAL\\Administrator"
	role_id = "${data.vsphere_role.default.role_id}"
	folder_path = "/"
	propagate = true
}
```

## Argument Reference

The following arguments are supported:

* `principal` - (Required) The name of the user/group.
* `role_id` - (Required) The role id that is associated to.
* `folder_path` - (Optional) The folder path that the entity applied permissions to. Default to "/"
* `propagate` - (Optional  Enable propagation to all the children folders
* `group` - (Optional) To mark the principal as group
