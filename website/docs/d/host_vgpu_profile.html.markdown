---
subcategory: "Host and Cluster Management"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_host_vgpu_profile"
sidebar_current: "docs-vsphere-data-source-host_vgpu_profile"
description: |-
  A data source that can be used to get information for one or all vGPU profiles
  available on an ESXi host.
---

# vsphere_host_vgpu_profile

The `vsphere_host_vgpu_profile` data source can be used to discover the
available vGPU profiles of a vSphere host.

## Example Usage to return all vGPU profiles

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc-01"
}

data "vsphere_host" "host" {
  name          = "esxi-01.example.com"
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

data "vsphere_host_vgpu_profile" "vgpu_profile" {
  host_id = data.vsphere_host.host.id
}
```

## Example Usage with vGPU profile name_regex

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc-01"
}

data "vsphere_host" "host" {
  name          = "esxi-01.example.com"
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

data "vsphere_host_vgpu_profile" "vgpu_profile" {
  host_id    = data.vsphere_host.host.id
  name_regex = "a100"
}
```

## Argument Reference

The following arguments are supported:

* `host_id` - (Required) The [managed object reference ID][docs-about-morefs] of
  a host.
* `name_regex` - (Optional) A regular expression that will be used to match the
  host vGPU profile name.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

## Attribute Reference

The following attributes are exported:

* `host_id` - The [managed objectID][docs-about-morefs] of the ESXi host.
* `id` - Unique (SHA256) id based on the host_id if the ESXi host.
* `name_regex` - (Optional) A regular expression that will be used to match the
  host vGPU profile name.
* `vgpu_profiles` - The list of available vGPU profiles on the ESXi host.
  This may be and empty array if no vGPU profile are identified.
  * `vgpu` - Name of a particular vGPU available as a shared GPU device (vGPU
    profile).
  * `disk_snapshot_supported` - Indicates whether the GPU plugin on this host is
    capable of disk-only snapshots when VM is not powered off.
  * `memory_snapshot_supported` - Indicates whether the GPU plugin on this host
    is capable of memory snapshots.
  * `migrate_supported` - Indicates whether the GPU plugin on this host is
    capable of migration.
  * `suspend_supported` - Indicates whether the GPU plugin on this host is
    capable of suspend-resume.
