---
subcategory: "Administration"
page_title: "VMware vSphere: vsphere_license"
sidebar_current: "docs-vsphere-resource-admin-license"
description: |-
  Provides a VMware vSphere license resource.
---

# vsphere_license

Provides a VMware vSphere license resource. This can be used to add and remove license keys.

## Example Usage

```hcl
resource "vsphere_license" "licenseKey" {
  license_key = "00000-00000-00000-00000-00000"

  labels = {
    VpxClientLicenseLabel = "example"
  }
}
```

## Argument Reference

The following arguments are supported:

* `license_key` - (Required) The license key value.
* `labels` - (Optional) A map of labels to be applied to the license key.

~> **NOTE:** Labels are not allowed for unmanaged ESX hosts.

## Attributes Reference

The following attributes are exported:

* `edition_key` - The product edition of the license key.
* `name` - The display name for the license key.
* `total` - The total number of units contained in the license key.
* `used` - The number of units assigned to this license key.

