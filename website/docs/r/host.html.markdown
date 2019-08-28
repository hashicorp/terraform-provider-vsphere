---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_host"
sidebar_current: "docs-vsphere-resource-inventory-host"
description: |-
  Provides a VMware vSphere host resource. This represents an ESXi host that can be used either as part of a Compute Cluster or Standalone.
---

# vsphere\_host

Provides a VMware vSphere host resource. This represents an ESXi host that
can be used either as part of a Compute Cluster or Standalone.

## Example Usages

**Create a standalone host:**

```hcl
data "vsphere_datacenter" "dc" {
  name = "my-datacenter"
}

resource "vsphere_host" "h1" {
  hostname = "10.10.10.1"
  username = "root"
  password = "password"
  license = "00000-00000-00000-00000i-00000"
  datacenter = data.vsphere_datacenter.dc.id
}
```

**Create host in a compute cluster:**

```hcl
data "vsphere_datacenter" "dc" {
  name = "TfDatacenter"
}

data "vsphere_compute_cluster" "c1" {
  name = "DC0_C0"
  datacenter_id = data.vsphere_datacenter.dc.id
}

resource "vsphere_host" "h1" {
  hostname = "10.10.10.1"
  username = "root"
  password = "password"
  license = "00000-00000-00000-00000i-00000"
  cluster = data.vsphere_compute_cluster.c1.id
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
  be added to. This should not be set if `datacenter` is set.
* `thumbprint` - (Optional) Host's certificate SHA-1 thumbprint. If not set the the
  CA that signed the host's certificate should be trusted. If the CA is not trusted
  and no thumbprint is set then the operation will fail.
* `license` - (Optional) The license key that will be applied to the host.
  The license key is expected to be present in vSphere.
* `force` - (Optional) If set to true then it will force the host to be added, even
  if the host is already connected to a different vSphere instance. Default is `false`
* `connected` - (Optional) If set to false then the host will be disconected.
  Default is `false`.
* `maintenance` - (Optional) Set the management state of the host. Default is `false`.
* `lockdown` - (Optional) Set the lockdown state of the host. Valid options are
  `disabled`, `normal`, and `strict`. Default is `disabled`.

## Attribute Reference

* `id` - The ID of the host.


## Importing 

An existing host can be [imported][docs-import] into this resource
via supplying the host's ID. An example is below:

[docs-import]: /docs/import/index.html

```
terraform import vsphere_host.vm host-123
```

The above would import the host with ID `host-123`.
