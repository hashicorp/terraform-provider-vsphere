---
subcategory: "Inventory"
page_title: "VMware vSphere: vsphere_tag_category"
sidebar_current: "docs-vsphere-resource-inventory-tag-category"
description: |-
  Provides a vSphere tag category resource. This can be used to manage tag categories in vSphere.
---

# vsphere_tag_category

The `vsphere_tag_category` resource can be used to create and manage tag
categories, which determine how tags are grouped together and applied to
specific objects.

For more information about tags, click [here][ext-tags-general].

[ext-tags-general]: https://techdocs.broadcom.com/us/en/vmware-cis/vsphere/vsphere/8-0/vsphere-tags-and-attributes.html

## Example Usage

This example creates a tag category named `terraform-test-category`, with
single cardinality (meaning that only one tag in this category can be assigned
to an object at any given time). Tags in this category can only be assigned to
VMs and datastores.

```hcl
resource "vsphere_tag_category" "category" {
  name        = "terraform-test-category"
  description = "Managed by Terraform"
  cardinality = "SINGLE"

  associable_types = [
    "VirtualMachine",
    "Datastore",
  ]
}
```

## Argument Reference

The following arguments are supported:

- `name` - (Required) The name of the category.
- `cardinality` - (Required) The number of tags that can be assigned from this
  category to a single object at once. Can be one of `SINGLE` (object can only
  be assigned one tag in this category), to `MULTIPLE` (object can be assigned
  multiple tags in this category). Forces a new resource if changed.
- `associable_types` - (Required) A list object types that this category is
  valid to be assigned to. For a full list, click
  [here](#associable-object-types).
- `description` - (Optional) A description for the category.

~> **NOTE:** You can add associable types to a category, but you cannot remove
them. Attempting to do so will result in an error.

### Associable Object Types

The following table will help you determine what values you need to enter for
the associable type you want to associate with a tag category.

Note that if you want a tag to apply to all objects you will need to add all the types listed below.

<table>
<tr><th>Type</th><th>Value</th></tr>
<tr><td>Folders</td><td>`Folder`</td></tr>
<tr><td>Clusters</td><td>`ClusterComputeResource`</td></tr>
<tr><td>Datacenters</td><td>`Datacenter`</td></tr>
<tr><td>Datastores</td><td>`Datastore`</td></tr>
<tr><td>Datastore Clusters</td><td>`StoragePod`</td></tr>
<tr><td>DVS Portgroups</td><td>`DistributedVirtualPortgroup`</td></tr>
<tr><td>Distributed vSwitches</td><td>`DistributedVirtualSwitch`<br>`VmwareDistributedVirtualSwitch`</td></tr>
<tr><td>Hosts</td><td>`HostSystem`</td></tr>
<tr><td>Content Libraries</td><td>`com.vmware.content.Library`</td></tr>
<tr><td>Content Library Items</td><td>`com.vmware.content.library.Item`</td></tr>
<tr><td>Networks</td><td>`HostNetwork`<br>`Network`<br>`OpaqueNetwork`</td></tr>
<tr><td>Resource Pools</td><td>`ResourcePool`</td></tr>
<tr><td>vApps</td><td>`VirtualApp`</td></tr>
<tr><td>Virtual Machines</td><td>`VirtualMachine`</td></tr>
</table>

## Attribute Reference

The only attribute that is exported for this resource is the `id`, which is the
uniform resource name (URN) of this tag category.

## Importing

An existing tag category can be [imported][docs-import] into this resource via
its name, using the following command:

[docs-import]: https://developer.hashicorp.com/terraform/cli/import

```shell
terraform import vsphere_tag_category.category terraform-test-category
```
