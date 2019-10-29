---
subcategory: "Storage"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_storage_drs_vm_override"
sidebar_current: "docs-vsphere-resource-storage-storage-drs-vm-override"
description: |-
  Provides a VMware vSphere Storage DRS virtual machine override resource. This can be used to override Storage DRS settings in a datastore cluster.
---

# vsphere\_storage\_drs\_vm\_override

The `vsphere_storage_drs_vm_override` resource can be used to add a Storage DRS
override to a datastore cluster for a specific virtual machine. With this
resource, one can enable or disable Storage DRS, and control the automation
level and disk affinity for a single virtual machine without affecting the rest
of the datastore cluster.

For more information on vSphere datastore clusters and Storage DRS, see [this
page][ref-vsphere-datastore-clusters].

[ref-vsphere-datastore-clusters]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.resmgmt.doc/GUID-598DF695-107E-406B-9C95-0AF961FC227A.html

## Example Usage

The example below builds on the [Storage DRS
example][tf-vsphere-vm-storage-drs-example] in the `vsphere_virtual_machine`
resource. However, rather than use the output of the
[`vsphere_datastore_cluster` data
source][tf-vsphere-datastore-cluster-data-source] for the location of the
virtual machine, we instead get what is assumed to be a member datastore using
the [`vsphere_datastore` data source][tf-vsphere-datastore-data-source] and put
the virtual machine there instead. We then use the
`vsphere_storage_drs_vm_override` resource to ensure that Storage DRS does not
apply to this virtual machine, and hence the VM will never be migrated off of
the datastore.

[tf-vsphere-vm-storage-drs-example]: /docs/providers/vsphere/r/virtual_machine.html#using-storage-drs
[tf-vsphere-datastore-cluster-data-source]: /docs/providers/vsphere/d/datastore_cluster.html
[tf-vsphere-datastore-data-source]: /docs/providers/vsphere/d/datastore.html

```hcl
data "vsphere_datacenter" "dc" {
  name = "dc1"
}

data "vsphere_datastore_cluster" "datastore_cluster" {
  name          = "datastore-cluster1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_datastore" "member_datastore" {
  name          = "datastore-cluster1-member1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "cluster1/Resources"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "public"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.member_datastore.id}"

  num_cpus = 2
  memory   = 1024
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }
}

resource "vsphere_storage_drs_vm_override" "drs_vm_override" {
  datastore_cluster_id = "${data.vsphere_datastore_cluster.datastore_cluster.id}"
  virtual_machine_id   = "${vsphere_virtual_machine.vm.id}"
  sdrs_enabled         = false
}
```

## Argument Reference

The following arguments are supported:

* `datastore_cluster_id` - (Required) The [managed object reference
  ID][docs-about-morefs] of the datastore cluster to put the override in.
  Forces a new resource if changed.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

* `virtual_machine_id` - (Required) The UUID of the virtual machine to create
  the override for.  Forces a new resource if changed.
* `sdrs_enabled` - (Optional) Overrides the default Storage DRS setting for
  this virtual machine. When not specified, the datastore cluster setting is
  used.
* `sdrs_automation_level` - (Optional) Overrides any Storage DRS automation
  levels for this virtual machine. Can be one of `automated` or `manual`. When
  not specified, the datastore cluster's settings are used according to the
  [specific SDRS subsystem][tf-vsphere-datastore-cluster-sdrs-levels].

[tf-vsphere-datastore-cluster-sdrs-levels]: /docs/providers/vsphere/r/datastore_cluster.html#storage-drs-automation-options

* `sdrs_intra_vm_affinity` - (Optional) Overrides the intra-VM affinity setting
  for this virtual machine. When `true`, all disks for this virtual machine
  will be kept on the same datastore. When `false`, Storage DRS may locate
  individual disks on different datastores if it helps satisfy cluster
  requirements. When not specified, the datastore cluster's settings are used.

## Attribute Reference

The only attribute this resource exports is the `id` of the resource, which is
a combination of the [managed object reference ID][docs-about-morefs] of the
datastore cluster, and the UUID of the virtual machine. This is used to look up
the override on subsequent plan and apply operations after the override has
been created.

## Importing

An existing override can be [imported][docs-import] into this resource by
supplying both the path to the datastore cluster and the path to the virtual
machine to `terraform import`. If no override exists, an error will be given.
An example is below:

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_storage_drs_vm_override.drs_vm_override \
  '{"datastore_cluster_path": "/dc1/datastore/ds-cluster", \
  "virtual_machine_path": "/dc1/vm/srv1"}'
```
