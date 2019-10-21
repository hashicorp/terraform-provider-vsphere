---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_storage_policy"
sidebar_current: "docs-vsphere-data-source-storage-policy"
description: |-
  A data source that can be used to get the UUID of a storage policy.
---

# vsphere\_storage\_policy

The `vsphere_storage_policy` data source can be used to discover the UUID of a
vSphere storage policy. This can then be used with resources or data sources that
require a storage policy.

~> **NOTE:** Storage policy support is unsupported on direct ESXi connections and
requires vCenter 6.0 or higher.

## Example Usage

```hcl
data "vsphere_storage_policy" "policy" {
  name = "policy1"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the storage policy.

## Attribute Reference

The only exported attribute is `id`, which is the UUID of this storage policy.

