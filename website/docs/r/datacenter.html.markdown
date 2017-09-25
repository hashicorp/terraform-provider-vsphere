---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_datacenter"
sidebar_current: "docs-vsphere-resource-inventory-datacenter"
description: |-
  Provides a VMware vSphere datacenter resource. This can be used as the primary container of inventory objects such as hosts and virtual machines.
---

# vsphere\_datacenter

Provides a VMware vSphere datacenter resource. This can be used as the primary
container of inventory objects such as hosts and virtual machines.

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

* `name` - (Required) The name of the datacenter. This name needs to be unique
  within the folder. Forces a new resource if changed.
* `folder` - (Optional) The folder where the datacenter should be created.
  Forces a new resource if changed.
* `tags` - (Optional) The IDs of any tags to attach to this resource. See
  [here][docs-applying-tags] for a reference on how to apply tags.

[docs-applying-tags]: /docs/providers/vsphere/r/tag.html#using-tags-in-a-supported-resource

~> **NOTE:** Tagging support is unsupported on direct ESXi connections and
requires vCenter 6.0 or higher.

## Attribute Reference

The only attribute exported is `id`, which is the name of the datacenter.
