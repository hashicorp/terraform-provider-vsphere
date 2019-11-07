---
subcategory: "Host and Cluster Management"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_resource_pool"
sidebar_current: "docs-vsphere-resource-compute-resource-pool"
description: |-
  Provides a vSphere resource pool resource. This can be used to create and manage resource pools.
---

# vsphere\_resource\_pool

The `vsphere_resource_pool` resource can be used to create and manage
resource pools in standalone hosts or on compute clusters.

For more information on vSphere resource pools, see [this
page][ref-vsphere-resource_pools].

[ref-vsphere-resource_pools]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.resmgmt.doc/GUID-60077B40-66FF-4625-934A-641703ED7601.html

## Example Usage

The following example sets up a resource pool in a compute cluster which uses
the default settings for CPU and memory reservations, shares, and limits. The
compute cluster needs to already exist in vSphere.  

```hcl
variable "datacenter" {
  default = "dc1"
}

variable "cluster" {
  default = "cluster1"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_compute_cluster" "compute_cluster" {
  name          = "${var.cluster}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_resource_pool" "resource_pool" {
  name                    = "terraform-resource-pool-test"
  parent_resource_pool_id = "${data.vsphere_compute_cluster.compute_cluster.resource_pool_id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the resource pool.
* `parent_resource_pool_id` - (Required) The [managed object ID][docs-about-morefs]
  of the parent resource pool. This can be the root resource pool for a cluster
  or standalone host, or a resource pool itself. When moving a resource pool
  from one parent resource pool to another, both must share a common root
  resource pool or the move will fail.
* `cpu_share_level` - (Optional) The CPU allocation level. The level is a
  simplified view of shares. Levels map to a pre-determined set of numeric
  values for shares. Can be one of `low`, `normal`, `high`, or `custom`. When
  `low`, `normal`, or `high` are specified values in `cpu_shares` will be
  ignored.  Default: `normal`
* `cpu_shares` - (Optional) The number of shares allocated for CPU. Used to
  determine resource allocation in case of resource contention. If this is set,
  `cpu_share_level` must be `custom`.
* `cpu_reservation` - (Optional) Amount of CPU (MHz) that is guaranteed
  available to the resource pool. Default: `0`
* `cpu_expandable` - (Optional) Determines if the reservation on a resource
  pool can grow beyond the specified value if the parent resource pool has
  unreserved resources. Default: `true`
* `cpu_limit` - (Optional) The CPU utilization of a resource pool will not exceed
  this limit, even if there are available resources. Set to `-1` for unlimited.
  Default: `-1`
* `memory_share_level` - (Optional) The CPU allocation level. The level is a
  simplified view of shares. Levels map to a pre-determined set of numeric
  values for shares. Can be one of `low`, `normal`, `high`, or `custom`. When
  `low`, `normal`, or `high` are specified values in `memory_shares` will be
  ignored.  Default: `normal`
* `memory_shares` - (Optional) The number of shares allocated for CPU. Used to
  determine resource allocation in case of resource contention. If this is set,
  `memory_share_level` must be `custom`.
* `memory_reservation` - (Optional) Amount of CPU (MHz) that is guaranteed
  available to the resource pool. Default: `0`
* `memory_expandable` - (Optional) Determines if the reservation on a resource
  pool can grow beyond the specified value if the parent resource pool has
  unreserved resources. Default: `true`
* `memory_limit` - (Optional) The CPU utilization of a resource pool will not exceed
  this limit, even if there are available resources. Set to `-1` for unlimited.
  Default: `-1`
* `tags` - (Optional) The IDs of any tags to attach to this resource. See
  [here][docs-applying-tags] for a reference on how to apply tags.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider
[docs-applying-tags]: /docs/providers/vsphere/r/tag.html#using-tags-in-a-supported-resource

## Attribute Reference

The only attribute this resource exports is the `id` of the resource, which is
the [managed object ID][docs-about-morefs] of the resource pool.

## Importing

An existing resource pool can be [imported][docs-import] into this resource via
the path to the resource pool, using the following command:

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_resource_pool.resource_pool /dc1/host/compute-cluster1/Resources/resource-pool1
```

The above would import the resource pool named `resource-pool1` that is located
in the compute cluster `compute-cluster1` in the `dc1` datacenter.
