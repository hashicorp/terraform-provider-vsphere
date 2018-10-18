---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_vapp_entity"
sidebar_current: "docs-vsphere-resource-compute-vapp-entity"
description: |-
  Provides a vSphere vApp entity resource. This can be used to describe the behavior of an entity (virtual machine or sub-vApp container) in a vApp container.
---

# vsphere_vapp_entity

The `vsphere_vapp_entity` resource can be used to describe the behavior of an
entity (virtual machine or sub-vApp container) in a vApp container.

For more information on vSphere vApps, see [this
page][ref-vsphere-vapp].

[ref-vsphere-vapp]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.vm_admin.doc/GUID-2A95EBB8-1779-40FA-B4FB-4D0845750879.html

## Example Usage

The basic example below sets up a vApp container and a virtual machine in a
compute cluster and then creates a vApp entity to change the virtual machine's
power on behavior in the vApp container.

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

resource "vsphere_vapp_entity" "vapp_entity" {
  target_id    = "vsphere_virtual_machine.vm.id"
  container_id = "vsphere_vapp_container.vapp_container.id"
  start_action = "non"
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

* `target_id` - (Required) [Managed object ID|docs-about-morefs] of the entity
  to power on or power off. This can be a virtual machine or a vApp. 
* `container_id` - (Required) [Managed object ID|docs-about-morefs] of the vApp
  container the entity is a member of.
* `start_order` - (Optional) Order to start and stop target in vApp. Default: 1
* `start_action` - (Optional) How to start the entity. Valid settings are none
  or powerOn. If set to none, then the entity does not participate in auto-start.
  Default: powerOn
* `start_delay` - (Optional) Delay in seconds before continuing with the next
  entity in the order of entities to be started. Default: 120 
* `stop_action` - (Optional) Defines the stop action for the entity. Can be set
  to none, powerOff, guestShutdown, or suspend. If set to none, then the entity
  does not participate in auto-stop. Default: powerOff 
* `stop_delay` - (Optional) Delay in seconds before continuing with the next
  entity in the order sequence. This is only used if the stopAction is
  guestShutdown. Default: 120 
* `wait_for_guest` - (Optional) Determines if the VM should be marked as being
  started when VMware Tools are ready instead of waiting for `start_delay`. This
  property has no effect for vApps. Default: false


[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

## Attribute Reference

The only attribute this resource exports is the `id` of the resource, which is
the vApp entity's [managed object ID][docs-about-morefs] separated from the
virtual machines [managed object ID][docs-about-morefs] by a colon.

## Importing

An existing vApp entity can be [imported][docs-import] into this resource via
the ID of the vApp Entity.

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_vapp_entity.vapp_entity vm-123:res-456 
```

The above would import the vApp entity that governs the behavior of the virtual
machine with a [managed object ID][docs-about-morefs] of vm-123 in the vApp
container with the [managed object ID][docs-about-morefs] res-456.
