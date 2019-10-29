---
subcategory: "Storage"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_datastore_cluster"
sidebar_current: "docs-vsphere-resource-storage-datastore-cluster"
description: |-
  Provides a vSphere datastore cluster resource. This can be used to create and manage datastore clusters.
---

# vsphere\_datastore\_cluster

The `vsphere_datastore_cluster` resource can be used to create and manage
datastore clusters. This can be used to create groups of datastores with a
shared management interface, allowing for resource control and load balancing
through Storage DRS.

For more information on vSphere datastore clusters and Storage DRS, see [this
page][ref-vsphere-datastore-clusters].

[ref-vsphere-datastore-clusters]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.resmgmt.doc/GUID-598DF695-107E-406B-9C95-0AF961FC227A.html

~> **NOTE:** This resource requires vCenter and is not available on direct ESXi
connections.

~> **NOTE:** Storage DRS requires a vSphere Enterprise Plus license.

## Example Usage

The following example sets up a datastore cluster and enables Storage DRS with
the default settings. It then creates two NAS datastores using the
[`vsphere_nas_datastore` resource][ref-tf-nas-datastore] and assigns them to
the datastore cluster.

[ref-tf-nas-datastore]: /docs/providers/vsphere/r/nas_datastore.html

```hcl
variable "hosts" {
  default = [
    "esxi1",
    "esxi2",
    "esxi3",
  ]
}

data "vsphere_datacenter" "datacenter" {}

data "vsphere_host" "esxi_hosts" {
  count         = "${length(var.hosts)}"
  name          = "${var.hosts[count.index]}"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_datastore_cluster" "datastore_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
  sdrs_enabled  = true
}

resource "vsphere_nas_datastore" "datastore1" {
  name                 = "terraform-datastore-test1"
  host_system_ids      = ["${data.vsphere_host.esxi_hosts.*.id}"]
  datastore_cluster_id = "${vsphere_datastore_cluster.datastore_cluster.id}"

  type         = "NFS"
  remote_hosts = ["nfs"]
  remote_path  = "/export/terraform-test1"
}

resource "vsphere_nas_datastore" "datastore2" {
  name                 = "terraform-datastore-test2"
  host_system_ids      = ["${data.vsphere_host.esxi_hosts.*.id}"]
  datastore_cluster_id = "${vsphere_datastore_cluster.datastore_cluster.id}"

  type         = "NFS"
  remote_hosts = ["nfs"]
  remote_path  = "/export/terraform-test2"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the datastore cluster.
* `datacenter_id` - (Required) The [managed object ID][docs-about-morefs] of
  the datacenter to create the datastore cluster in. Forces a new resource if
  changed.
* `folder` - (Optional) The relative path to a folder to put this datastore
  cluster in.  This is a path relative to the datacenter you are deploying the
  datastore to.  Example: for the `dc1` datacenter, and a provided `folder` of
  `foo/bar`, Terraform will place a datastore cluster named
  `terraform-datastore-cluster-test` in a datastore folder located at
  `/dc1/datastore/foo/bar`, with the final inventory path being
  `/dc1/datastore/foo/bar/terraform-datastore-cluster-test`.
* `sdrs_enabled` - (Optional) Enable Storage DRS for this datastore cluster.
  Default: `false`.
* `tags` - (Optional) The IDs of any tags to attach to this resource. See
  [here][docs-applying-tags] for a reference on how to apply tags.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider
[docs-applying-tags]: /docs/providers/vsphere/r/tag.html#using-tags-in-a-supported-resource

~> **NOTE:** Tagging support requires vCenter 6.0 or higher.

* `custom_attributes` - (Optional) A map of custom attribute ids to attribute
  value strings to set for the datastore cluster. See
  [here][docs-setting-custom-attributes] for a reference on how to set values
  for custom attributes.

[docs-setting-custom-attributes]: /docs/providers/vsphere/r/custom_attribute.html#using-custom-attributes-in-a-supported-resource

~> **NOTE:** Custom attributes are unsupported on direct ESXi connections 
and require vCenter.

### Storage DRS automation options

The following options control the automation levels for Storage DRS on the
datastore cluster.

All options below can either be one of two settings: `manual` for manual mode,
where Storage DRS makes migration recommendations but does not execute them, or
`automated` for fully automated mode, where Storage DRS executes migration
recommendations automatically.

The automation level can be further tuned for each specific SDRS subsystem.
Specifying an override will set the automation level for that part of Storage
DRS to the respective level. Not specifying an override infers that you want to
use the cluster default automation level.

* `sdrs_automation_level` - (Optional) The global automation level for all
  virtual machines in this datastore cluster. Default: `manual`.
* `sdrs_space_balance_automation_level` - (Optional) Overrides the default
  automation settings when correcting disk space imbalances.
* `sdrs_io_balance_automation_level` - (Optional) Overrides the default
  automation settings when correcting I/O load imbalances.
* `sdrs_rule_enforcement_automation_level` - (Optional) Overrides the default
  automation settings when correcting affinity rule violations.
* `sdrs_policy_enforcement_automation_level` - (Optional) Overrides the default
  automation settings when correcting storage and VM policy violations.
* `sdrs_vm_evacuation_automation_level` - (Optional) Overrides the default
  automation settings when generating recommendations for datastore evacuation.

### Storage DRS I/O load balancing settings

The following options control I/O load balancing for Storage DRS on the
datastore cluster.

~> **NOTE:** All reservable IOPS settings require vSphere 6.0 or higher and are
ignored on older versions.

* `sdrs_io_load_balance_enabled` - (Optional) Enable I/O load balancing for
  this datastore cluster. Default: `true`.
* `sdrs_io_latency_threshold` - (Optional) The I/O latency threshold, in
  milliseconds, that storage DRS uses to make recommendations to move disks
  from this datastore. Default: `15` seconds.
* `sdrs_io_load_imbalance_threshold` - (Optional) The difference between load
  in datastores in the cluster before storage DRS makes recommendations to
  balance the load. Default: `5` percent.
* `sdrs_io_reservable_iops_threshold` - (Optional) The threshold of reservable
  IOPS of all virtual machines on the datastore before storage DRS makes
  recommendations to move VMs off of a datastore. Note that this setting should
  only be set if `sdrs_io_reservable_percent_threshold` cannot make an accurate
  estimate of the capacity of the datastores in your cluster, and should be set
  to roughly 50-60% of the worst case peak performance of the backing LUNs.
* `sdrs_io_reservable_percent_threshold` - (Optional) The threshold, in
  percent, of actual estimated performance of the datastore (in IOPS) that
  storage DRS uses to make recommendations to move VMs off of a datastore when
  the total reservable IOPS exceeds the threshold. Default: `60` percent.
* `sdrs_io_reservable_threshold_mode` - (Optional) The reservable IOPS
  threshold setting to use, `sdrs_io_reservable_percent_threshold` in the event
  of `automatic`, or `sdrs_io_reservable_iops_threshold` in the event of
  `manual`. Default: `automatic`.

### Storage DRS disk space load balancing settings

The following options control disk space load balancing for Storage DRS on the
datastore cluster.

~> **NOTE:** Setting `sdrs_free_space_threshold_mode` to `freeSpace` and using
the `sdrs_free_space_threshold` setting requires vSphere 6.0 or higher and is
ignored on older versions. Using these settings on older versions may result in
spurious diffs in Terraform.

* `sdrs_free_space_utilization_difference` - (Optional) The threshold, in
  percent of used space, that storage DRS uses to make decisions to migrate VMs
  out of a datastore. Default: `80` percent.
* `sdrs_free_space_utilization_difference` - (Optional) The threshold, in
  percent, of difference between space utilization in datastores before storage
  DRS makes decisions to balance the space. Default: `5` percent.
* `sdrs_free_space_threshold` - (Optional) The threshold, in GB, that storage
  DRS uses to make decisions to migrate VMs out of a datastore. Default: `50`
  GB.
* `sdrs_free_space_threshold` - (Optional) The free space threshold to use.
  When set to `utilization`, `drs_space_utilization_threshold` is used, and
  when set to `freeSpace`, `drs_free_space_threshold` is used. Default:
  `utilization`.

### Storage DRS advanced settings

The following options control advanced parts of Storage DRS that may not
require changing during basic operation:

* `sdrs_default_intra_vm_affinity` - (Optional) When `true`, all disks in a
  single virtual machine will be kept on the same datastore. Default: `true`.
* `sdrs_load_balance_interval` - (Optional) The storage DRS poll interval, in
  minutes. Default: `480` minutes.
* `sdrs_advanced_options` - (Optional) A key/value map of advanced Storage DRS
  settings that are not exposed via Terraform or the vSphere client.

## Attribute Reference

The only computed attribute that is exported by this resource is the resource
`id`, which is the the [managed object reference ID][docs-about-morefs] of the
datastore cluster.

## Importing

An existing datastore cluster can be [imported][docs-import] into this resource
via the path to the cluster, via the following command:

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_datastore_cluster.datastore_cluster /dc1/datastore/ds-cluster
```

The above would import the datastore cluster named `ds-cluster` that is located
in the `dc1` datacenter.
