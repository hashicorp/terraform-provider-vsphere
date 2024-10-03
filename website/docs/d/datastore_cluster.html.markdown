---
subcategory: "Storage"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_datastore_cluster"
sidebar_current: "docs-vsphere-data-source-cluster-datastore"
description: |-
  Provides a data source to return the ID of a vSphere datastore cluster object.
---

# vsphere\_datastore\_cluster

The `vsphere_datastore_cluster` data source can be used to discover the ID of a
vSphere datastore cluster object. This can then be used with resources or data
sources that require a datastore. For example, to assign datastores using the
[`vsphere_nas_datastore`][docs-nas-datastore-resource] or
[`vsphere_vmfs_datastore`][docs-vmfs-datastore-resource] resources, or to create
virtual machines in using the
[`vsphere_virtual_machine`][docs-virtual-machine-resource] resource.

[docs-nas-datastore-resource]: /docs/providers/vsphere/r/nas_datastore.html
[docs-vmfs-datastore-resource]: /docs/providers/vsphere/r/vmfs_datastore.html
[docs-virtual-machine-resource]: /docs/providers/vsphere/r/virtual_machine.html

## Example Usage

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc-01"
}

data "vsphere_datastore_cluster" "datastore_cluster" {
  name          = "datastore-cluster-01"
  datacenter_id = data.vsphere_datacenter.datacenter.id
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name or absolute path to the datastore cluster.
* `datacenter_id` - (Optional) The [managed object reference
  ID][docs-about-morefs] of the datacenter the datastore cluster is located in.
  This can be omitted if the search path used in `name` is an absolute path.
  For default datacenters, use the id attribute from an empty
  `vsphere_datacenter` data source.


[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

## Attribute Reference

The following attributes are exported:

* `id` - The [managed objectID][docs-about-morefs] of the vSphere datastore cluster object.

* `datastores` - (Optional) The names of the datastores included in the specific 
  cluster.

### Example Usage for `datastores` attribute:

```hcl
output "datastores" {
  value = data.vsphere_datastore_cluster.datastore_cluster.datastores
}
```
