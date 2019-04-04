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
data "vsphere_datacenter" "dc" {
  name = "dc1"
}

data "vsphere_datastore" "ds" {
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
  name          = "nfsds1"
}

data "vsphere_role" "default" {
	name = "Admin"
}

resource "vsphere_entity_permission" "default" {
	principal   = "VSPHERE.LOCAL\\Administrator"
	role_id     = "${data.vsphere_role.default.role_id}"
  entity_id   = "${data.vsphere_datastore.ds.id}"
  entity_type = "Datastore"
	propagate   = true
  group       = false
}
```

## Argument Reference

The following arguments are supported:

* `principal` - (Required) The name of the user/group.
* `entity_id` - (Required) The [managed object ID][docs-about-morefs] of the entity to apply the permission to.
* `entity_type` - (Required) The [type][ref-vsphere-moid-types] of entity to apply the permission to.
* `role_id` - (Optional) The role ID that is associated with the specified principal and entity.
* `propagate` - (Optional) Determines if the entity permission should propagate to children of the specified entity.
* `group` - (Optional) Specifies the principal to use is a group. Default: `false`

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider
[ref-vsphere-moid-types]: https://pubs.vmware.com/vsphere-50/index.jsp?topic=%2Fcom.vmware.wssdk.apiref.doc_50%2Fvmodl.ManagedObjectReference.html

