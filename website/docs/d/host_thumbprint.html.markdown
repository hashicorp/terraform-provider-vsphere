---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_host_thumbprint"
sidebar_current: "docs-vsphere-data-source-datacenter"
description: |-
  A data source that can be used to get the thumbprint of an ESXi host.
---

# vsphere\_host\_thumbprint

The `vsphere_thumbprint` data source can be used to discover the host
thumbprint of an ESXi host. This can be used when adding the `vsphere_host`
resource. If the host is using a certificate chain, the first one returned
will be used to generate the thumbprint.

## Example Usage

```hcl
data "vsphere_host_thumbprint" "thumbprint" {
  address = "esxi.example.internal"
}
```

## Argument Reference

The following arguments are supported:

* `address` - (Required) The address of the ESXi host to retrieve the
thumbprint from.
* `port` - (Optional) The port to use connecting to the ESXi host. Default: 443
* `insecure` - (Optional) Boolean that can be set to true to disable SSL 
certificate verification. Default: false

## Attribute Reference

The only exported attribute is `id`, which is the thumbprint of the ESXi
host.