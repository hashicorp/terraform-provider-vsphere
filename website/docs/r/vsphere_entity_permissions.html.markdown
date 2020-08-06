---
subcategory: "Administration Resources"
layout: "vsphere"
page_title: "VMware vSphere: Entity Permissions"
sidebar_current: "docs-vsphere-entity-permissions"
description: |-
  Provides CRUD operations on a vsphere entity permissions. Permissions can be created on an entity for a given user or 
  group with the specified roles.
---

# vsphere\_entity\_permissions

The `vsphere_entity_permissions` resource can be used to create and manage entity permissions. 
Permissions can be created on an entity for a given user or group with the specified role.

## Example Usage

This example creates entity permissions on the virtual machine VM1 for the user group DCClients with role Datastore 
consumer and for user group ExternalIDPUsers with role my_terraform_role. The `entity_id` can be the managed object id
(or uuid for some resources). The `entity_type` is one of the vmware managed object types which can be found from the 
managed object types section in [vmware_api_7](https://code.vmware.com/apis/968/vsphere). Keep the permissions sorted
alphabetically on `user_or_group` for a better user experience.


```hcl

data "vsphere_datacenter" "dc" {
  name = "Sample_DC_2"
}

data "vsphere_virtual_machine" vm1 {
  name = "VM1"
  datacenter_id = data.vsphere_datacenter.dc.id
}

data "vsphere_role" "role1" {
  label = "Datastore consumer (sample)"
}

resource vsphere_role "role2" {
  name = "my_terraform_role"
  role_privileges = ["Alarm.Acknowledge", "Alarm.Create", "Datacenter.Move"]
}

resource "vsphere_entity_permissions" p1 {
  entity_id = data.vsphere_virtual_machine.vm1.id
  entity_type = "VirtualMachine"
  permissions {
    user_or_group = "vsphere.local\\DCClients"
    propagate = true
    is_group = true
    role_id = data.vsphere_role.role1.id
  }
  permissions {
    user_or_group = "vsphere.local\\ExternalIDPUsers"
    propagate = true
    is_group = true
    role_id = vsphere_role.role2.id
  }
}

```

## Argument Reference

The following arguments are supported:

* `entity_id`   - (Required) The managed object id (uuid for some entities) on which permissions are to be created.
* `entity_type` - (Required) The managed object type, types can be found in the managed object type section 
   [here](https://code.vmware.com/apis/968/vsphere).

* `permissions`     - (Required) The permissions to be given on this entity. Keep the permissions sorted
                       alphabetically on `user_or_group` for a better user experience.
  * `user_or_group` - (Required) The user/group getting the permission.
  * `is_group`      - (Required) Whether user_or_group field refers to a user or a group. True for a group and false for a user.
  * `role_id`       - (Required) The role id of the role to be given to the user on the specified entity.
  * `propagate`     - (Required) Whether or not this permission propagates down the hierarchy to sub-entities.

