---
subcategory: "Host and Cluster Management"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_drs_vm_override"
sidebar_current: "docs-vsphere-resource-compute-drs-vm-override"
description: |-
  Provides a VMware vSphere DRS virtual machine override resource. This can be used to override DRS settings in a cluster.
---

# vsphere\_drs\_vm\_override

The `vsphere_drs_vm_override` resource can be used to add a DRS override to a
cluster for a specific virtual machine. With this resource, one can enable or
disable DRS and control the automation level for a single virtual machine
without affecting the rest of the cluster.

For more information on vSphere clusters and DRS, see [this
page][ref-vsphere-drs-clusters].

[ref-vsphere-drs-clusters]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.resmgmt.doc/GUID-8ACF3502-5314-469F-8CC9-4A9BD5925BC2.html

~> **NOTE:** This resource requires vCenter and is not available on direct ESXi
connections.

~> **NOTE:** vSphere DRS requires a vSphere Enterprise Plus license.

## Example Usage

The example below creates a virtual machine in a cluster using the
[`vsphere_virtual_machine`][tf-vsphere-vm-resource] resource, creating the
virtual machine in the cluster looked up by the
[`vsphere_compute_cluster`][tf-vsphere-cluster-data-source] data source, but also
pinning the VM to a host defined by the
[`vsphere_host`][tf-vsphere-host-data-source] data source, which is assumed to
be a host within the cluster. To ensure that the VM stays on this host and does
not need to be migrated back at any point in time, an override is entered using
the `vsphere_drs_vm_override` resource that disables DRS for this virtual
machine, ensuring that it does not move.

[tf-vsphere-vm-resource]: /docs/providers/vsphere/r/virtual_machine.html
[tf-vsphere-cluster-data-source]: /docs/providers/vsphere/d/compute_cluster.html
[tf-vsphere-host-data-source]: /docs/providers/vsphere/d/host.html

```hcl
data "vsphere_datacenter" "dc" {
  name = "dc1"
}

data "vsphere_datastore" "datastore" {
  name          = "datastore1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "cluster1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_host" "host" {
  name          = "esxi1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "network1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
  host_system_id   = "${data.vsphere_host.host.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }
}

resource "vsphere_drs_vm_override" "drs_vm_override" {
  compute_cluster_id = "${data.vsphere_compute_cluster.cluster.id}"
  virtual_machine_id = "${vsphere_virtual_machine.vm.id}"
  drs_enabled        = false
}
```

## Argument Reference

The following arguments are supported:

* `compute_cluster_id` - (Required) The [managed object reference
  ID][docs-about-morefs] of the cluster to put the override in.  Forces a new
  resource if changed.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

* `virtual_machine_id` - (Required) The UUID of the virtual machine to create
  the override for.  Forces a new resource if changed.
* `drs_enabled` - (Optional) Overrides the default DRS setting for this virtual
  machine. Can be either `true` or `false`. Default: `false`.
* `drs_automation_level` - (Optional) Overrides the automation level for this virtual
  machine in the cluster. Can be one of `manual`, `partiallyAutomated`, or
  `fullyAutomated`. Default: `manual`.

-> **NOTE:** Using this resource _always_ implies an override, even if one of
`drs_enabled` or `drs_automation_level` is omitted. Take note of the defaults
for both options.

## Attribute Reference

The only attribute this resource exports is the `id` of the resource, which is
a combination of the [managed object reference ID][docs-about-morefs] of the
cluster, and the UUID of the virtual machine. This is used to look up the
override on subsequent plan and apply operations after the override has been
created.

## Importing

An existing override can be [imported][docs-import] into this resource by
supplying both the path to the cluster, and the path to the virtual machine, to
`terraform import`. If no override exists, an error will be given.  An example
is below:

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_drs_vm_override.drs_vm_override \
  '{"compute_cluster_path": "/dc1/host/cluster1", \
  "virtual_machine_path": "/dc1/vm/srv1"}'
```
