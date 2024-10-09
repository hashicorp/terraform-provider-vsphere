---
subcategory: "Host and Cluster Management"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_host_thumbprint"
sidebar_current: "docs-vsphere-data-source-datacenter"
description: |-
  A data source that can be used to get the thumbprint of an ESXi host.
---

# vsphere\_host\_thumbprint

The `vsphere_thumbprint` data source can be used to discover the host thumbprint
of an ESXi host. This can be used when adding the `vsphere_host` resource to a
cluster or a vCenter Server instance.

* If the ESXi host is using a certificate chain, the first one returned will be
used to generate the thumbprint.

* If the ESXi host has a certificate issued by a certificate authority, ensure
that the the certificate authority is trusted on the system running the plan.

## Example Usage

```hcl
data "vsphere_host_thumbprint" "thumbprint" {
  address = "esxi-01.example.com"
}
```

## Argument Reference

The following arguments are supported:

* `address` - (Required) The address of the ESXi host to retrieve the thumbprint
  from.
* `insecure` - (Optional) Disables SSL certificate verification. Default: `false`
* `port` - (Optional) The port to use connecting to the ESXi host. Default: 443

## Attribute Reference

The only exported attribute is `id`, which is the thumbprint of the ESXi host.
