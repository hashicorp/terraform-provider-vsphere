---
subcategory: "Inventory"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_custom_attribute"
sidebar_current: "docs-vsphere-resource-inventory-custom-attribute"
description: |-
  Provides a vSphere custom attribute resource. This can be used to manage custom attributes in vSphere.
---

# vsphere\_custom\_attribute

The `vsphere_custom_attribute` resource can be used to create and manage custom
attributes, which allow users to associate user-specific meta-information with 
vSphere managed objects. Custom attribute values must be strings and are stored 
on the vCenter Server and not the managed object.

For more information about custom attributes, click [here][ext-custom-attributes].

[ext-custom-attributes]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.vcenterhost.doc/GUID-73606C4C-763C-4E27-A1DA-032E4C46219D.html

~> **NOTE:** Custom attributes are unsupported on direct ESXi connections 
and require vCenter.

## Example Usage

This example creates a custom attribute named `terraform-test-attribute`. The 
resulting custom attribute can be assigned to VMs only.

```hcl
resource "vsphere_custom_attribute" "attribute" {
  name                = "terraform-test-attribute"
  managed_object_type = "VirtualMachine"
}
```

## Using Custom Attributes in a Supported Resource

Custom attributes can be set on vSphere resources in Terraform via the 
`custom_attributes` argument in any supported resource.

The following example builds on the above example by creating a 
[`vsphere_virtual_machine`][docs-virtual-machine-resource] and assigning a 
value to created custom attribute on it.

[docs-virtual-machine-resource]: /docs/providers/vsphere/r/virtual_machine.html

```hcl
resource "vsphere_custom_attribute" "attribute" {
  name                = "terraform-test-attribute"
  managed_object_type = "VirtualMachine"
}

resource "vpshere_virtual_machine" "web" {
  # ... other configuration ...

  custom_attributes = "${map(vsphere_custom_attribute.attribute.id, "value")}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the custom attribute.
* `managed_object_type` - (Optional) The object type that this attribute may be
  applied to. If not set, the custom attribute may be applied to any object
  type. For a full list, click [here](#managed-object-types). Forces a new
  resource if changed.

## Managed Object Types

The following table will help you determine what value you need to enter for 
the managed object type you want the attribute to apply to.

Note that if you want a attribute to apply to all objects, leave the type 
unspecified.

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

This resource only exports the `id` attribute for the vSphere custom attribute.

## Importing

An existing custom attribute can be [imported][docs-import] into this resource 
via its name, using the following command:

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_custom_attribute.attribute terraform-test-attribute
```
