---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_datacenter"
sidebar_current: "docs-vsphere-resource-inventory-datacenter"
description: |-
  Provides a VMware vSphere datacenter resource. This can be used as the primary container of inventory objects such as hosts and virtual machines.
---

# vsphere\_datacenter

Provides a VMware vSphere datacenter resource. This can be used as the primary container of inventory objects such as hosts and virtual machines.

## Example Usages

**Create datacenter on the root folder:**

```hcl
resource "vsphere_datacenter" "prod_datacenter" {
  name       = "my_prod_datacenter"
}
```

**Create datacenter on a subfolder:**

```hcl
resource "vsphere_datacenter" "research_datacenter" {
  name       = "my_research_datacenter"
  folder     = "/research/"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the datacenter. This name needs to be unique within the folder.
* `folder` - (Optional) The folder where the datacenter should be created.

~> **NOTE**: Datacenters cannot be changed once they are created. Modifying any of these attributes will force a new resource!
