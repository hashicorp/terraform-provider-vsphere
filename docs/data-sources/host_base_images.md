---
subcategory: "Lifecycle"
page_title: "VMware vSphere: vsphere_host_base_images"
sidebar_current: "docs-vsphere-data-source-host-base-images"
description: |-
  Provides a VMware vSphere ESXi base images data source.
  This can be used to get the list of ESXi base images available for cluster
  software management.
---

# vsphere_host_base_images

The `vsphere_host_base_images` data source can be used to get the list of ESXi
base images available for cluster software management.

## Example Usage

```hcl
data "vsphere_host_base_images" "base_images" {}
```

## Attribute Reference

The following attributes are exported:

* `base_images` - List of available images.
  * `version` - The ESXi version identifier for the image
