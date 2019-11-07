---
subcategory: "Storage"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_vmfs_datastore"
sidebar_current: "docs-vsphere-resource-storage-vmfs-datastore"
description: |-
  Provides a vSphere VMFS datastore resource. This can be used to configure a VMFS datastore on a host or set of hosts.
---

# vsphere\_vmfs\_datastore

The `vsphere_vmfs_datastore` resource can be used to create and manage VMFS
datastores on an ESXi host or a set of hosts. The resource supports using any
SCSI device that can generally be used in a datastore, such as local disks, or
disks presented to a host or multiple hosts over Fibre Channel or iSCSI.
Devices can be specified manually, or discovered using the
[`vsphere_vmfs_disks`][data-source-vmfs-disks] data source.

[data-source-vmfs-disks]: /docs/providers/vsphere/d/vmfs_disks.html 

## Auto-Mounting of Datastores Within vCenter

Note that the current behaviour of this resource will auto-mount any created
datastores to any other host within vCenter that has access to the same disk.

Example: You want to create a datastore with a iSCSI LUN that is visible on 3
hosts in a single vSphere cluster (`esxi1`, `esxi2` and `esxi3`). When you
create the datastore on `esxi1`, the datastore will be automatically mounted on
`esxi2` and `esxi3`, without the need to configure the resource on either of
those two hosts.

Future versions of this resource may allow you to control the hosts that a
datastore is mounted to, but currently, this automatic behaviour cannot be
changed, so keep this in mind when writing your configurations and deploying
your disks.

## Increasing Datastore Size

To increase the size of a datastore, you must add additional disks to the
`disks` attribute. Expanding the size of a datastore by increasing the size of
an already provisioned disk is currently not supported (but may be in future
versions of this resource).

~> **NOTE:** You cannot decrease the size of a datastore. If the resource
detects disks removed from the configuration, Terraform will give an error. To
reduce the size of the datastore, the resource needs to be re-created - run
[`terraform taint`][cmd-taint] to taint the resource so it can be re-created.

[cmd-taint]: /docs/commands/taint.html

## Example Usage

### Addition of local disks on a single host

The following example uses the default datacenter and default host to add a
datastore with local disks to a single ESXi server.

~> **NOTE:** There are some situations where datastore creation will not work
when working through vCenter (usually when trying to create a datastore on a
single host with local disks). If you experience trouble creating the datastore
you need through vCenter, break the datstore off into a different configuration
and deploy it using the ESXi server as the provider endpoint, using a similar
configuration to what is below.

```hcl
data "vsphere_datacenter" "datacenter" {}

data "vsphere_host" "esxi_host" {
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_vmfs_datastore" "datastore" {
  name           = "terraform-test"
  host_system_id = "${data.vsphere_host.esxi_host.id}"

  disks = [
    "mpx.vmhba1:C0:T1:L0",
    "mpx.vmhba1:C0:T2:L0",
    "mpx.vmhba1:C0:T2:L0",
  ]
}
```

### Auto-detection of disks via `vsphere_vmfs_disks`

The following example makes use of the
[`vsphere_vmfs_disks`][data-source-vmfs-disks] data source to auto-detect
exported iSCSI LUNS matching a certain NAA vendor ID (in this case, LUNs
exported from a [NetApp][ext-netapp]). These discovered disks are then loaded
into `vsphere_vmfs_datastore`. The datastore is also placed in the
`datastore-folder` folder afterwards.

[ext-netapp]: https://kb.netapp.com/support/s/article/ka31A0000000rLRQAY/how-to-match-a-lun-s-naa-number-to-its-serial-number?language=en_US

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc1"
}

data "vsphere_host" "esxi_host" {
  name          = "esxi1"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

data "vsphere_vmfs_disks" "available" {
  host_system_id = "${data.vsphere_host.esxi_host.id}"
  rescan         = true
  filter         = "naa.60a98000"
}

resource "vsphere_vmfs_datastore" "datastore" {
  name           = "terraform-test"
  host_system_id = "${data.vsphere_host.esxi_host.id}"
  folder         = "datastore-folder"

  disks = ["${data.vsphere_vmfs_disks.available.disks}"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the datastore. Forces a new resource if
  changed.
* `host_system_id` - (Required) The [managed object ID][docs-about-morefs] of
  the host to set the datastore up on. Note that this is not necessarily the
  only host that the datastore will be set up on - see
  [here](#auto-mounting-of-datastores-within-vcenter) for more info. Forces a
  new resource if changed.
* `disks` - (Required) The disks to use with the datastore.
* `folder` - (Optional) The relative path to a folder to put this datastore in.
  This is a path relative to the datacenter you are deploying the datastore to.
  Example: for the `dc1` datacenter, and a provided `folder` of `foo/bar`,
  Terraform will place a datastore named `terraform-test` in a datastore folder
  located at `/dc1/datastore/foo/bar`, with the final inventory path being
  `/dc1/datastore/foo/bar/terraform-test`. Conflicts with
  `datastore_cluster_id`.
* `datastore_cluster_id` - (Optional) The [managed object
  ID][docs-about-morefs] of a datastore cluster to put this datastore in.
  Conflicts with `folder`.
* `tags` - (Optional) The IDs of any tags to attach to this resource. See
  [here][docs-applying-tags] for a reference on how to apply tags.

[docs-applying-tags]: /docs/providers/vsphere/r/tag.html#using-tags-in-a-supported-resource
[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

~> **NOTE:** Tagging support is unsupported on direct ESXi connections and
requires vCenter 6.0 or higher.

* `custom_attributes` (Optional) Map of custom attribute ids to attribute 
   value string to set on datastore resource. See 
   [here][docs-setting-custom-attributes] for a reference on how to set values 
   for custom attributes.

[docs-setting-custom-attributes]: /docs/providers/vsphere/r/custom_attribute.html#using-custom-attributes-in-a-supported-resource

~> **NOTE:** Custom attributes are unsupported on direct ESXi connections 
and require vCenter.

## Attribute Reference

The following attributes are exported:

* `id` - The [managed object reference ID][docs-about-morefs] of the datastore.
* `accessible` - The connectivity status of the datastore. If this is `false`,
  some other computed attributes may be out of date.
* `capacity` - Maximum capacity of the datastore, in megabytes.
* `free_space` - Available space of this datastore, in megabytes.
* `maintenance_mode` - The current maintenance mode state of the datastore.
* `multiple_host_access` - If `true`, more than one host in the datacenter has
  been configured with access to the datastore.
* `uncommitted_space` - Total additional storage space, in megabytes,
  potentially used by all virtual machines on this datastore.
* `url` - The unique locator for the datastore.

## Importing

An existing VMFS datastore can be [imported][docs-import] into this resource
via its managed object ID, via the command below. You also need the host system
ID.

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_vmfs_datastore.datastore datastore-123:host-10
```

You need a tool like [`govc`][ext-govc] that can display managed object IDs.

[ext-govc]: https://github.com/vmware/govmomi/tree/master/govc

In the case of govc, you can locate a managed object ID from an inventory path
by doing the following:

```
$ govc ls -i /dc/datastore/terraform-test
Datastore:datastore-123
```

To locate host IDs, it might be a good idea to supply the `-l` flag as well so
that you can line up the names with the IDs:

```
$ govc ls -l -i /dc/host/cluster1
ResourcePool:resgroup-10 /dc/host/cluster1/Resources
HostSystem:host-10 /dc/host/cluster1/esxi1
HostSystem:host-11 /dc/host/cluster1/esxi2
HostSystem:host-12 /dc/host/cluster1/esxi3
```
