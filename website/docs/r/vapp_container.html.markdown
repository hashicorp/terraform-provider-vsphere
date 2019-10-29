---
subcategory: "Virtual Machine"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_vapp_container"
sidebar_current: "docs-vsphere-resource-compute-vapp-container"
description: |-
  Provides a vSphere vApp container resource. This can be used to create and manage vApp container.
---

# vsphere\_vapp\_container

The `vsphere_vapp_container` resource can be used to create and manage
vApps.

For more information on vSphere vApps, see [this
page][ref-vsphere-vapp].

[ref-vsphere-vapp]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.vm_admin.doc/GUID-2A95EBB8-1779-40FA-B4FB-4D0845750879.html

## Example Usage

The basic example below sets up a vApp container in a compute cluster which uses
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

resource "vsphere_vapp_container" "vapp_container" {
  name                    = "terraform-vapp-container-test"
  parent_resource_pool_id = "${data.vsphere_compute_cluster.compute_cluster.id}"
}
```

### Example with virtual machine

The below example builds off the basic example, but includes a virtual machine
in the new vApp container. To accomplish this, the `resource_pool_id` of the
virtual machine is set to the `id` of the vApp container.

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

data "vsphere_network" "network" {
  name          = "network1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_datastore" "datastore" {
  name          = "datastore1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_vapp_container" "vapp_container" {
  name                    = "terraform-vapp-container-test"
  parent_resource_pool_id = "${data.vsphere_compute_cluster.compute_cluster.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-virutal-machine-test"
  resource_pool_id = "${vsphere_vapp_container.vapp_container.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"
  num_cpus         = 2
  memory           = 1024
  guest_id         = "ubuntu64Guest"

  disk {
    label = "disk0"
    size  = 1
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the vApp container.
* `parent_resource_pool_id` - (Required) The [managed object ID][docs-about-morefs]
  of the parent resource pool. This can be the root resource pool for a cluster
  or standalone host, or a resource pool itself. When moving a vApp container
  from one parent resource pool to another, both must share a common root
  resource pool or the move will fail.
* `parent_folder_id` - (Optional) The [managed object ID][docs-about-morefs] of
  the vApp container's parent folder.
* `cpu_share_level` - (Optional) The CPU allocation level. The level is a
  simplified view of shares. Levels map to a pre-determined set of numeric
  values for shares. Can be one of `low`, `normal`, `high`, or `custom`. When
  `low`, `normal`, or `high` are specified values in `cpu_shares` will be
  ignored.  Default: `normal`
* `cpu_shares` - (Optional) The number of shares allocated for CPU. Used to
  determine resource allocation in case of resource contention. If this is set,
  `cpu_share_level` must be `custom`.
* `cpu_reservation` - (Optional) Amount of CPU (MHz) that is guaranteed
  available to the vApp container. Default: `0`
* `cpu_expandable` - (Optional) Determines if the reservation on a vApp
  container can grow beyond the specified value if the parent resource pool has
  unreserved resources. Default: `true`
* `cpu_limit` - (Optional) The CPU utilization of a vApp container will not
  exceed this limit, even if there are available resources. Set to `-1` for
  unlimited.
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
  available to the vApp container. Default: `0`
* `memory_expandable` - (Optional) Determines if the reservation on a vApp
  container can grow beyond the specified value if the parent resource pool has
  unreserved resources. Default: `true`
* `memory_limit` - (Optional) The CPU utilization of a vApp container will not
  exceed this limit, even if there are available resources. Set to `-1` for
  unlimited.
  Default: `-1`
* `tags` - (Optional) The IDs of any tags to attach to this resource. See
  [here][docs-applying-tags] for a reference on how to apply tags.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider
[docs-applying-tags]: /docs/providers/vsphere/r/tag.html#using-tags-in-a-supported-resource

## Attribute Reference

The only attribute this resource exports is the `id` of the resource, which is
the [managed object ID][docs-about-morefs] of the resource pool.

## Importing

An existing vApp container can be [imported][docs-import] into this resource via
the path to the vApp container, using the following command:

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_vapp_container.vapp_container /default-dc/host/cluster1/Resources/resource_pool1/vapp_container1
```

The above would import the vApp container named `vapp-container1` that is
located in the resource pool `resource-pool1` that is part of the host cluster
`cluster1` in the `dc1` datacenter.
