---
subcategory: "Host and Cluster Management"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_dpm_host_override"
sidebar_current: "docs-vsphere-resource-compute-dpm-host-override"
description: |-
  Provides a VMware vSphere DPM host override resource. This can be used to override power management settings for a host in a cluster.
---

# vsphere\_dpm\_host\_override

The `vsphere_dpm_host_override` resource can be used to add a DPM override to a
cluster for a particular host. This allows you to control the power management
settings for individual hosts in the cluster while leaving any unspecified ones
at the default power management settings.

For more information on DPM within vSphere clusters, see [this
page][ref-vsphere-cluster-dpm].

[ref-vsphere-cluster-dpm]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.resmgmt.doc/GUID-5E5E349A-4644-4C9C-B434-1C0243EBDC80.html

~> **NOTE:** This resource requires vCenter and is not available on direct ESXi
connections.

~> **NOTE:** vSphere DRS requires a vSphere Enterprise Plus license.

## Example Usage

The following example creates a compute cluster comprised of three hosts,
making use of the
[`vsphere_compute_cluster`][tf-vsphere-compute-cluster-resource] resource. DPM
will be disabled in the cluster as it is the default setting, but we override
the setting of the first host referenced by the
[`vsphere_host`][tf-vsphere-host-data-source] data source (`esxi1`) by using
the `vsphere_dpm_host_override` resource so it will be powered off when the
cluster does not need it to service virtual machines.

[tf-vsphere-compute-cluster-resource]: /docs/providers/vsphere/r/compute_cluster.html
[tf-vsphere-host-data-source]: /docs/providers/vsphere/d/host.html

```hcl
variable "datacenter" {
  default = "dc1"
}

variable "hosts" {
  default = [
    "esxi1",
    "esxi2",
    "esxi3",
  ]
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_host" "hosts" {
  count         = "${length(var.hosts)}"
  name          = "${var.hosts[count.index]}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name            = "terraform-compute-cluster-test"
  datacenter_id   = "${data.vsphere_datacenter.dc.id}"
  host_system_ids = ["${data.vsphere_host.hosts.*.id}"]

  drs_enabled          = true
  drs_automation_level = "fullyAutomated"
}

resource "vsphere_dpm_host_override" "dpm_host_override" {
  compute_cluster_id   = "${vsphere_compute_cluster.compute_cluster.id}"
  host_system_id       = "${data.vsphere_host.hosts.0.id}"
  dpm_enabled          = true
  dpm_automation_level = "automated"
}
```

## Argument Reference

The following arguments are supported:

* `compute_cluster_id` - (Required) The [managed object reference
  ID][docs-about-morefs] of the cluster to put the override in.  Forces a new
  resource if changed.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

* `host_system_ids` - (Optional) The [managed object ID][docs-about-morefs] of
  the host to create the override for.
* `dpm_enabled` - (Optional) Enable DPM support for this host. Default:
  `false`. 
* `dpm_automation_level` - (Optional) The automation level for host power
  operations on this host. Can be one of `manual` or `automated`. Default:
  `manual`.

-> **NOTE:** Using this resource _always_ implies an override, even if one of
`dpm_enabled` or `dpm_automation_level` is omitted. Take note of the defaults
for both options.

## Attribute Reference

The only attribute this resource exports is the `id` of the resource, which is
a combination of the [managed object reference ID][docs-about-morefs] of the
cluster, and the managed object reference ID of the host. This is used to look
up the override on subsequent plan and apply operations after the override has
been created.

## Importing

An existing override can be [imported][docs-import] into this resource by
supplying both the path to the cluster, and the path to the host, to `terraform
import`. If no override exists, an error will be given.  An example is below:

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_dpm_host_override.dpm_host_override \
  '{"compute_cluster_path": "/dc1/host/cluster1", \
  "host_path": "/dc1/host/esxi1"}'
```
