---
subcategory: "Administration"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_license"
sidebar_current: "docs-vsphere-data-source-admin-license"
description: |-
  Provides a VMware vSphere license data source. This can be used to get the
  general attributes of license keys.
---

# vsphere\_license

The `vsphere_license` data source can be used to get the general attributes of
a license keys from a vCenter Server instance.

## Example Usage

```hcl
data "vsphere_license" "license" {
  license_key = "00000-00000-00000-00000-00000"
}
```

## Argument Reference

The following arguments are supported:

* `license_key` - (Required) The license key.

## Attribute Reference

The following attributes are exported:

* `labels` - A map of key/value pairs attached as labels (tags) to the license key.
* `edition_key` - The product edition of the license key.
* `total` - Total number of units (example: CPUs) contained in the license.
* `used` - The number of units (example: CPUs) assigned to this license.
* `name` - The display name for the license.
