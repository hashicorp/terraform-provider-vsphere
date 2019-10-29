---
subcategory: "Host and Cluster Management"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_ha_vm_override"
sidebar_current: "docs-vsphere-resource-compute-ha-vm-override"
description: |-
  Provides a VMware vSphere HA virtual machine override resource. This can be used to override high availability settings in a cluster.
---

# vsphere\_ha\_vm\_override

The `vsphere_ha_vm_override` resource can be used to add an override for
vSphere HA settings on a cluster for a specific virtual machine. With this
resource, one can control specific HA settings so that they are different than
the cluster default, accommodating the needs of that specific virtual machine,
while not affecting the rest of the cluster.

For more information on vSphere HA, see [this page][ref-vsphere-ha-clusters].

[ref-vsphere-ha-clusters]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.avail.doc/GUID-5432CA24-14F1-44E3-87FB-61D937831CF6.html

~> **NOTE:** This resource requires vCenter and is not available on direct ESXi
connections.

## Example Usage

The example below creates a virtual machine in a cluster using the
[`vsphere_virtual_machine`][tf-vsphere-vm-resource] resource, creating the
virtual machine in the cluster looked up by the
[`vsphere_compute_cluster`][tf-vsphere-cluster-data-source] data source.

Considering a scenario where this virtual machine is of high value to the
application or organization for which it does its work, it's been determined in
the event of a host failure, that this should be one of the first virtual
machines to be started by vSphere HA during recovery. Hence, its
[`ha_vm_restart_priority`](#ha_vm_restart_priority) as been set to `highest`,
which, assuming that the default restart priority is `medium` and no other
virtual machine has been assigned the `highest` priority, will mean that this
VM will be started before any other virtual machine in the event of host
failure.

[tf-vsphere-vm-resource]: /docs/providers/vsphere/r/virtual_machine.html
[tf-vsphere-cluster-data-source]: /docs/providers/vsphere/d/compute_cluster.html

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

data "vsphere_network" "network" {
  name          = "network1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
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

resource "vsphere_ha_vm_override" "ha_vm_override" {
  compute_cluster_id = "${data.vsphere_compute_cluster.cluster.id}"
  virtual_machine_id = "${vsphere_virtual_machine.vm.id}"

  ha_vm_restart_priority = "highest"
}
```

## Argument Reference

The following arguments are supported:

### General Options

The following options are required:

* `compute_cluster_id` - (Required) The [managed object reference
  ID][docs-about-morefs] of the cluster to put the override in.  Forces a new
  resource if changed.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

* `virtual_machine_id` - (Required) The UUID of the virtual machine to create
  the override for.  Forces a new resource if changed.

### vSphere HA Options

The following settings work nearly in the same fashion as their counterparts in
the [`vsphere_compute_cluster`][tf-vsphere-cluster-resource] resource, with the
exception that some options also allow settings that denote the use of cluster
defaults. See the individual settings below for more details.

[tf-vsphere-cluster-resource]: /docs/providers/vsphere/r/compute_cluster.html

~> **NOTE:** The same version restrictions that apply for certain options
within [`vsphere_compute_cluster`][tf-vsphere-cluster-resource] apply to
overrides as well. See [here][tf-vsphere-cluster-resource-version-restrictions]
for an entire list of version restrictions. 

[tf-vsphere-cluster-resource-version-restrictions]: /docs/providers/vsphere/r/compute_cluster.html#vsphere-version-requirements

#### General HA options

* `ha_vm_restart_priority` - (Optional) The restart priority for the virtual
  machine when vSphere detects a host failure. Can be one of
  `clusterRestartPriority`, `lowest`, `low`, `medium`, `high`, or `highest`.
  Default: `clusterRestartPriority`.
* `ha_vm_restart_timeout` - (Optional) The maximum time, in seconds, that
  vSphere HA will wait for this virtual machine to be ready. Use `-1` to
  specify the cluster default.  Default: `-1`.
  <sup>[\*][tf-vsphere-cluster-resource-version-restrictions]</sup>
* `ha_host_isolation_response` - (Optional) The action to take on this virtual
  machine when a host has detected that it has been isolated from the rest of
  the cluster. Can be one of `clusterIsolationResponse`, `none`, `powerOff`, or
  `shutdown`. Default: `clusterIsolationResponse`.

#### HA Virtual Machine Component Protection settings

The following settings control Virtual Machine Component Protection (VMCP)
overrides.

* `ha_datastore_pdl_response` - (Optional) Controls the action to take on this
  virtual machine when the cluster has detected a permanent device loss to a
  relevant datastore. Can be one of `clusterDefault`, `disabled`, `warning`, or
  `restartAggressive`. Default: `clusterDefault`.
  <sup>[\*][tf-vsphere-cluster-resource-version-restrictions]</sup>
* `ha_datastore_apd_response` - (Optional) Controls the action to take on this
  virtual machine when the cluster has detected loss to all paths to a relevant
  datastore. Can be one of `clusterDefault`, `disabled`, `warning`,
  `restartConservative`, or `restartAggressive`.  Default: `clusterDefault`.
  <sup>[\*][tf-vsphere-cluster-resource-version-restrictions]</sup>
* `ha_datastore_apd_recovery_action` - (Optional) Controls the action to take
  on this virtual machine if an APD status on an affected datastore clears in
  the middle of an APD event. Can be one of `useClusterDefault`, `none` or
  `reset`.  Default: `useClusterDefault`.
  <sup>[\*][tf-vsphere-cluster-resource-version-restrictions]</sup>
* `ha_datastore_apd_response_delay` - (Optional) Controls the delay in minutes
  to wait after an APD timeout event to execute the response action defined in
  [`ha_datastore_apd_response`](#ha_datastore_apd_response). Use `-1` to use
  the cluster default. Default: `-1`.
  <sup>[\*][tf-vsphere-cluster-resource-version-restrictions]</sup>

#### HA virtual machine and application monitoring settings

The following settings control virtual machine and application monitoring
overrides.

-> Take note of the
[`ha_vm_monitoring_use_cluster_defaults`](#ha_vm_monitoring_use_cluster_defaults)
setting - this is defaulted to `true` and means that override settings are
_not_ used. Set this to `false` to ensure your overrides function. Note that
unlike the rest of the options in this resource, there are no granular
per-setting cluster default values - `ha_vm_monitoring_use_cluster_defaults` is
the only toggle available.

* `ha_vm_monitoring_use_cluster_defaults` - (Optional) Determines whether or
  not the cluster's default settings or the VM override settings specified in
  this resource are used for virtual machine monitoring. The default is `true`
  (use cluster defaults) - set to `false` to have overrides take effect.
* `ha_vm_monitoring` - (Optional) The type of virtual machine monitoring to use
  when HA is enabled in the cluster. Can be one of `vmMonitoringDisabled`,
  `vmMonitoringOnly`, or `vmAndAppMonitoring`. Default: `vmMonitoringDisabled`.
* `ha_vm_failure_interval` - (Optional) If a heartbeat from this virtual
  machine is not received within this configured interval, the virtual machine
  is marked as failed. The value is in seconds. Default: `30`.
* `ha_vm_minimum_uptime` - (Optional) The time, in seconds, that HA waits after
  powering on this virtual machine before monitoring for heartbeats. Default:
  `120` (2 minutes).
* `ha_vm_maximum_resets` - (Optional) The maximum number of resets that HA will
  perform to this virtual machine when responding to a failure event. Default:
  `3`
* `ha_vm_maximum_failure_window` - (Optional) The length of the reset window in
  which [`ha_vm_maximum_resets`](#ha_vm_maximum_resets) can operate. When this
  window expires, no more resets are attempted regardless of the setting
  configured in `ha_vm_maximum_resets`. `-1` means no window, meaning an
  unlimited reset time is allotted. The value is specified in seconds. Default:
  `-1` (no window).

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
terraform import vsphere_ha_vm_override.ha_vm_override \
  '{"compute_cluster_path": "/dc1/host/cluster1", \
  "virtual_machine_path": "/dc1/vm/srv1"}'
```
