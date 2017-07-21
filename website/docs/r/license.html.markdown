---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_license"
sidebar_current: "docs-vsphere-resource-license"
description: |-
  Provides a VMware vSphere license resource. This can be used to add and remove license keys.
---

# vsphere\_license

Provides a VMware vSphere license resource. This can be used to add and remove license keys.

## Example Usage

```hcl
resource "vsphere_license" "licenseKey" {
  license_key = "452CQ-2EK54-K8742-00000-00000"

  label {
    key = "VpxClientLicenseLabel"
    value ="Hello World"
  }
  label {
    key = "Workflow"
    value ="Hello World"
  }
  
}
```

## Argument Reference

The following arguments are supported:

* `license_key` - (Required) The key of the license which is needed to be added to vpshere
* `label` - (Optional) The key value pair of labels that has to be attached to the license key. One key can have multiple labels.


## Attributes Reference

The following attributes are exported:

* `edition_key` - The product edition of the license key
* `total` - Total numbers of VCPUs supported by the license
* `used` - Total VCPUs currently being used 
* `name` - Name of the license