---
subcategory: "Storage"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_storage_policy"
sidebar_current: "docs-vsphere-data-source-storage-policy"
description: |-
  A data source that can be used to get the UUID of a storage policy.
---

# vsphere\_storage\_policy

The `vsphere_storage_policy` data source can be used to discover the UUID of a
storage policy. This can then be used with other resources or data sources that
use a storage policy.

~> **NOTE:** Storage policies are not supported on direct ESXi hosts and
requires vCenter Server.

## Example Usage

```hcl
data "vsphere_storage_policy" "prod_platinum_replicated" {
  name = "prod_platinum_replicated"
}

data "vsphere_storage_policy" "dev_silver_nonreplicated" {
  name = "dev_silver_nonreplicated"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the storage policy.

## Attribute Reference

The only exported attribute is `id`, which is the UUID of this storage policy.
