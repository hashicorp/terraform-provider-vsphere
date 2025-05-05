---
subcategory: "Inventory"
page_title: "VMware vSphere: vsphere_folder"
sidebar_current: "docs-vsphere-data-source-inventory-folder"
description: |-
  Provides a VMware vSphere folder data source. This can be used to get the
  general attributes of a vSphere inventory folder.
---

# vsphere_folder

The `vsphere_folder` data source can be used to get the general attributes of a
vSphere inventory folder. The data source supports creating folders of the 5
major types - datacenter folders, host and cluster folders, virtual machine
folders, storage folders, and network folders.

Paths are absolute and must include the datacenter.

## Example Usage

```hcl
resource "vsphere_folder" "datacenter_folder" {
  path = "example-datacenter-folder"
  type = "datacenter"
}

data "vsphere_folder" "datacenter_folder" {
  path       = "/${vsphere_folder.datacenter_folder.path}"
  depends_on = [vsphere_folder.datacenter_folder]
}

resource "vsphere_datacenter" "datacenter" {
  name       = "example-datacenter"
  folder     = data.vsphere_folder.datacenter_folder.path
  depends_on = [data.vsphere_folder.datacenter_folder]
}

data "vsphere_datacenter" "datacenter" {
  name       = vsphere_datacenter.datacenter.name
  depends_on = [vsphere_datacenter.datacenter]
}

resource "vsphere_folder" "vm_folder" {
  path          = "example-vm-folder"
  type          = "vm"
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

resource "vsphere_folder" "datastore_folder" {
  path          = "example-datastore-folder"
  type          = "datastore"
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

resource "vsphere_folder" "network_folder" {
  path          = "example-network-folder"
  type          = "network"
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

resource "vsphere_folder" "host_folder" {
  path          = "example-host-folder"
  type          = "host"
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

data "vsphere_folder" "vm_folder" {
  path       = "/${vsphere_folder.datacenter_folder.path}/${vsphere_datacenter.datacenter.name}/vm/${vsphere_folder.vm_folder.path}"
  depends_on = [vsphere_folder.vm_folder]
}

data "vsphere_folder" "datastore_folder" {
  path       = "/${vsphere_folder.datacenter_folder.path}/${vsphere_datacenter.datacenter.name}/datastore/${vsphere_folder.datastore_folder.path}"
  depends_on = [vsphere_folder.datastore_folder]
}

data "vsphere_folder" "network_folder" {
  path       = "/${vsphere_folder.datacenter_folder.path}/${vsphere_datacenter.datacenter.name}/network/${vsphere_folder.network_folder.path}"
  depends_on = [vsphere_folder.network_folder]
}

data "vsphere_folder" "host_folder" {
  path       = "/${vsphere_folder.datacenter_folder.path}/${vsphere_datacenter.datacenter.name}/host/${vsphere_folder.host_folder.path}"
  depends_on = [vsphere_folder.host_folder]
}

output "vm_folder_id" {
  value = data.vsphere_folder.vm_folder.id
}

output "datastore_folder_id" {
  value = data.vsphere_folder.datastore_folder.id
}

output "network_folder_id" {
  value = data.vsphere_folder.network_folder.id
}

output "host_folder_id" {
  value = data.vsphere_folder.host_folder.id
}

output "datacenter_id" {
  value = data.vsphere_datacenter.datacenter.id
}

output "datacenter_folder_path" {
  value = vsphere_folder.datacenter_folder.path
}
```

## Argument Reference

The following arguments are supported:

* `path` - (Required) The absolute path of the folder. For example, given a
  default datacenter of `default-dc`, a folder of type `vm`, and a folder name
  of `example-vm-folder`, the resulting `path` would be
  `/default-dc/vm/example-vm-folder`. 
  
  For nested datacenters, include the full hierarchy in the path. For example, if datacenter
  `default-dc` is inside folder `parent-folder`, the path to a VM folder would be
  `/parent-folder/default-dc/vm/example-vm-folder`.
  
  The valid folder types to be used in a `path` are: `vm`, `host`, `datacenter`, `datastore`, or `network`.
  
  Always include a leading slash in the `path`.

## Attribute Reference

The only attribute that this resource exports is the `id`, which is set to the
[managed object ID][docs-about-morefs] of the folder.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider
