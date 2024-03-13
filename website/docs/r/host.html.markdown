---
subcategory: "Host and Cluster Management"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_host"
sidebar_current: "docs-vsphere-resource-compute-host"
description: |-
  Provides a VMware vSphere host resource. This represents an ESXi host that
  can be used as a member of a cluster or as a standalone host.
---

# vsphere\_host

Provides a VMware vSphere host resource. This represents an ESXi host that
can be used either as a member of a cluster or as a standalone host.

## Example Usages

**Create a standalone host:**

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc-01"
}

data "vsphere_host_thumbprint" "thumbprint" {
  address = "esx-01.example.com"
  insecure = true
}

resource "vsphere_host" "esx-01" {
  hostname = "esx-01.example.com"
  username   = "root"
  password   = "password"
  license    = "00000-00000-00000-00000-00000"
  thumbprint = data.vsphere_host_thumbprint.thumbprint.id
  datacenter = data.vsphere_datacenter.datacenter.id
}
```

**Create host in a compute cluster:**

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc-01"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "cluster-01"
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

data "vsphere_host_thumbprint" "thumbprint" {
  address = "esx-01.example.com"
  insecure = true
}

resource "vsphere_host" "esx-01" {
  hostname = "esx-01.example.com"
  username = "root"
  password = "password"
  license  = "00000-00000-00000-00000-00000"
  thumbprint = data.vsphere_host_thumbprint.thumbprint.id
  cluster  = data.vsphere_compute_cluster.cluster.id
}
```

## Argument Reference

The following arguments are supported:

* `hostname` - (Required) FQDN or IP address of the host to be added.
* `username` - (Required) Username that will be used by vSphere to authenticate
  to the host.
* `password` - (Required) Password that will be used by vSphere to authenticate
  to the host.
* `datacenter` - (Optional) The ID of the datacenter this host should
  be added to. This should not be set if `cluster` is set.
* `cluster` - (Optional) The ID of the Compute Cluster this host should
  be added to. This should not be set if `datacenter` is set. Conflicts with:
  `cluster_managed`.
* `cluster_managed` - (Optional) Can be set to `true` if compute cluster
  membership will be managed through the `compute_cluster` resource rather
  than the`host` resource. Conflicts with: `cluster`.
* `thumbprint` - (Optional) Host's certificate SHA-1 thumbprint. If not set the
  CA that signed the host's certificate should be trusted. If the CA is not
  trusted and no thumbprint is set then the operation will fail. See data source
  [`vsphere_host_thumbprint`][docs-host-thumbprint-data-source].
* `license` - (Optional) The license key that will be applied to the host.
  The license key is expected to be present in vSphere.
* `force` - (Optional) If set to `true` then it will force the host to be added,
  even if the host is already connected to a different vCenter Server instance.
  Default is `false`.
* `connected` - (Optional) If set to false then the host will be disconnected.
  Default is `false`.
* `maintenance` - (Optional) Set the management state of the host.
  Default is `false`.
* `lockdown` - (Optional) Set the lockdown state of the host. Valid options are
  `disabled`, `normal`, and `strict`. Default is `disabled`.
* `tags` - (Optional) The IDs of any tags to attach to this resource. Please
  refer to the `vsphere_tag` resource for more information on applying
  tags to resources.

~> **NOTE:** Tagging support is not supported on direct ESXi host
connections and require vCenter Server.

* `custom_attributes` - (Optional) A map of custom attribute IDs and string
  values to apply to the resource. Please refer to the
  `vsphere_custom_attributes` resource for more information on applying
  tags to resources.

~> **NOTE:** Custom attributes are not supported on direct ESXi host
connections and require vCenter Server.

[docs-host-thumbprint-data-source]: /docs/providers/vsphere/d/host_thumbprint.html

## Attribute Reference

* `id` - The ID of the host.

## Importing

An existing host can be [imported][docs-import] into this resource by supplying
the host's ID. An example is below:

[docs-import]: /docs/import/index.html

```
terraform import vsphere_host.esx-01 host-123
```

The above would import the host with ID `host-123`.
