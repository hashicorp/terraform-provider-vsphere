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
data "vsphere_datacenter" "dc" {
  name = "dc1"
}

data "vsphere_datastore" "ds" {
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
  name          = "nfsds1"
}

data "vsphere_entity_permission" "admin" {
  principal   = "VSPHERE.LOCAL\\Administrator"
  entity_id   = "#{data.vsphere_datastore.ds.id}"
  entity_type = "Datastore"
}
```

## Argument Reference

* `principal` - (Required) The name of the user/group.
* `entity_id` - (Required) The [managed object ID][docs-about-morefs] of the entity to apply the permission to.
* `entity_type` - (Required) The [type][ref-vsphere-moid-types] of entity to apply the permission to.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider
[ref-vsphere-moid-types]: https://pubs.vmware.com/vsphere-50/index.jsp?topic=%2Fcom.vmware.wssdk.apiref.doc_50%2Fvmodl.ManagedObjectReference.html

## Attribute Reference

* `role_id` - The role ID that is associated with the specified principal and entity.
