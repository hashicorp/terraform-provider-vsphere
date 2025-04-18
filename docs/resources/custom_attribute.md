---
subcategory: "Inventory"
page_title: "VMware vSphere: vsphere_custom_attribute"
sidebar_current: "docs-vsphere-resource-inventory-custom-attribute"
description: |-
  Provides a VMware vSphere custom attribute resource. This can be used to manage custom attributes in vSphere.
---

# vsphere_custom_attribute

The `vsphere_custom_attribute` resource can be used to create and manage custom
attributes, which allow users to associate user-specific meta-information with
vSphere managed objects. Custom attribute values must be strings and are stored
on the vCenter Server and not the managed object.

For more information about custom attributes, click [here][ext-custom-attributes].

[ext-custom-attributes]: https://techdocs.broadcom.com/us/en/vmware-cis/vsphere/vsphere/8-0/vcenter-and-host-management-8-0/vsphere-tags-and-attributes-host-management/custom-attributes-in-the-vsphere-client-host-management.html

~> **NOTE:** Custom attributes are unsupported on direct ESXi host connections
and require vCenter Server.

## Example Usage

This example creates a custom attribute named `terraform-test-attribute`. The
resulting custom attribute can be assigned to VMs only.

```hcl
resource "vsphere_custom_attribute" "attribute" {
  name                = "terraform-test-attribute"
  managed_object_type = "VirtualMachine"
}
```

## Using Custom Attributes on a Supported Resource

Custom attributes can be set on supported provider resources using the
`custom_attributes` argument.

The following example creates both a `vsphere_custom_attribute` resource and a
[`vsphere_virtual_machine`][docs-virtual-machine-resource] resource. The custom attribute is then applied with an assigned value to the virtual machine.

[docs-virtual-machine-resource]: /docs/providers/vsphere/r/virtual_machine.html

```hcl
data "vsphere_custom_attribute" "attribute" {
  depends_on = [
    vsphere_custom_attribute.attribute
  ]
  name = "Owner"
}

resource "vsphere_custom_attribute" "attribute" {
  name                = "Owner"
  managed_object_type = "VirtualMachine"
}

resource "vsphere_virtual_machine" "vm" {
  # ... other configuration ...
  custom_attributes = tomap({ data.vsphere_custom_attribute.attribute.id = "John Doe" })
  # ... other configuration ...
}
```

The following example creates a [`vsphere_virtual_machine`][docs-virtual-machine-resource] resource and an existing custom attribute is then applied with an assigned value to the virtual machine.

[docs-virtual-machine-resource]: /docs/providers/vsphere/r/virtual_machine.html

```hcl
data "vsphere_custom_attribute" "attribute" {
  name = "Owner"
}

resource "vsphere_virtual_machine" "vm" {
  # ... other configuration ...
  custom_attributes = tomap({ data.vsphere_custom_attribute.attribute.id = "John Doe" })
  # ... other configuration ...
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the custom attribute.
* `managed_object_type` - (Optional) The object type that this attribute may be
  applied to. If not set, the custom attribute may be applied to any object
  type. For a full list, review the [Managed Object Types](#managed-object-types). Forces a new resource if changed.

## Managed Object Types

The following table provides the managed object values for which an attribute may apply.

~> **Note:** If you want an attribute to apply to all objects, leave the type unspecified and it will be global.

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

[docs-import]: https://developer.hashicorp.com/terraform/cli/import

```shell
terraform import vsphere_custom_attribute.attribute terraform-test-attribute
```
