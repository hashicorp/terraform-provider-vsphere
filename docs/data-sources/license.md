---
subcategory: "Administration"
page_title: "VMware vSphere: vsphere_license"
sidebar_current: "docs-vsphere-data-source-admin-license"
description: |-
  Provides a VMware vSphere license data source.
---

# vsphere_license

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

* `license_key` - (Required) The license key value.

## Attribute Reference

The following attributes are exported:

* `id` - The license key ID.
* `labels` - A map of labels applied to the license key.
* `edition_key` - The product edition of the license key.
* `name` - The display name for the license.
* `total` - The total number of units contained in the license key.
* `used` - The number of units assigned to this license key.

~> **NOTE:** Labels are not available for unmanaged ESX hosts.
