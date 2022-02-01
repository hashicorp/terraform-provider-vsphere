---
subcategory: "Networking"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_distributed_port_group"
sidebar_current: "docs-vsphere-resource-networking-distributed-port-group"
description: |-
  Provides a vSphere distributed port group resource. This can be used to
  create and manage port groups on a vSphere Distributed Switch.
---

# vsphere\_distributed\_port\_group

The `vsphere_distributed_port_group` resource can be used to manage
distributed port groups connected to vSphere Distributed Switches (VDS).
A vSphere Distributed Switch can be managed by the
[`vsphere_distributed_virtual_switch`][distributed-virtual-switch] resource.

Distributed port groups can be used as networks for virtual machines, allowing
the virtual machines to use the networking supplied by a vSphere Distributed
Switch, with a set of policies that apply to that individual network, if
desired.

* For an overview on vSphere networking concepts, refer to the vSphere  
  [product documentation][ref-vsphere-net-concepts].

* For more information on distributed port groups, refer to the vSphere
  [product documentation][ref-vsphere-dvportgroup].

[distributed-virtual-switch]: /docs/providers/vsphere/r/distributed_virtual_switch.html
[ref-vsphere-net-concepts]: https://docs.vmware.com/en/VMware-vSphere/7.0/com.vmware.vsphere.networking.doc/GUID-2B11DBB8-CB3C-4AFF-8885-EFEA0FC562F4.html
[ref-vsphere-dvportgroup]: https://docs.vmware.com/en/VMware-vSphere/7.0/com.vmware.vsphere.networking.doc/GUID-69933F6E-2442-46CF-AA17-1196CB9A0A09.html

~> **NOTE:** This resource requires vCenter Server and is not available on
direct ESXi host connections.

## Example Usage

The configuration below builds on the example given in the
[`vsphere_distributed_virtual_switch`][distributed-virtual-switch] resource by
adding the `vsphere_distributed_port_group` resource, attaching itself to the
vSphere Distributed Switch and assigning VLAN ID 1000.

```hcl
variable "esxi_hosts" {
  default = [
    "esxi-01.example.com",
    "esxi-02.example.com",
    "esxi-03.example.com",
  ]
}

variable "network_interfaces" {
  default = [
    "vmnic0",
    "vmnic1",
    "vmnic2",
    "vmnic3",
  ]
}

data "vsphere_datacenter" "datacenter" {
  name = "dc-01"
}

data "vsphere_host" "host" {
  count         = length(var.esxi_hosts)
  name          = var.esxi_hosts[count.index]
  datacenter_id = data.vsphere_datacenter.datacenter.id
}

resource "vsphere_distributed_virtual_switch" "vds" {
  name          = "vds-01"
  datacenter_id = data.vsphere_datacenter.datacenter.id

  uplinks         = ["uplink1", "uplink2", "uplink3", "uplink4"]
  active_uplinks  = ["uplink1", "uplink2"]
  standby_uplinks = ["uplink3", "uplink4"]

  host {
    host_system_id = data.vsphere_host.host.0.id
    devices        = ["${var.network_interfaces}"]
  }

  host {
    host_system_id = data.vsphere_host.host.1.id
    devices        = ["${var.network_interfaces}"]
  }

  host {
    host_system_id = data.vsphere_host.host.2.id
    devices        = ["${var.network_interfaces}"]
  }
}

resource "vsphere_distributed_port_group" "pg" {
  name                            = "pg-01"
  distributed_virtual_switch_uuid = vsphere_distributed_virtual_switch.vds.id

  vlan_id = 1000
}
```

### VLAN options

The following options control the VLAN setting. One of these 3 options may be set:

- `vlan_id` - (Optional) The member VLAN for the ports this policy applies to. A
  value of `0` means no VLAN.
- `vlan_range` - (Optional) Used to denote VLAN trunking. Use the `min_vlan`
  and `max_vlan` sub-arguments to define the tagged VLAN range. Multiple
  `vlan_range` definitions are allowed, but they must not overlap. Example
  below:

```hcl
resource "vsphere_distributed_virtual_switch" "vds" {
  name                            = "pg-01"
  distributed_virtual_switch_uuid = vsphere_distributed_virtual_switch.vds.id

  vlan_range {
    min_vlan = 100
    max_vlan = 199
  }
  vlan_range {
    min_vlan = 300
    max_vlan = 399
  }
}
```
- `port_private_secondary_vlan_id` - (Optional) Used to define a secondary VLAN
  ID when using private VLANs.

### Overriding VDS policies

All of the [default port policies][vds-default-port-policies] available in the
`vsphere_distributed_virtual_switch` resource can be overridden on the port
group level by specifying new settings for them.

[vds-default-port-policies]: /docs/providers/vsphere/r/distributed_virtual_switch.html#default-port-group-policy-arguments

As an example, we also take this example from the
`vsphere_distributed_virtual_switch` resource where we manually specify our
uplink count and uplink order. While the vSphere Distributed Switch has a
default policy of using the first uplink as an active uplink and the second
one as a standby, the overridden port group policy means that both uplinks
will be used as active uplinks in this specific port group.

```hcl
resource "vsphere_distributed_virtual_switch" "vds" {
  name          = "vds-01"
  datacenter_id = data.vsphere_datacenter.datacenter.id

  uplinks         = ["uplink1", "uplink2"]
  active_uplinks  = ["uplink1"]
  standby_uplinks = ["uplink2"]
}

resource "vsphere_distributed_port_group" "pg" {
  name                            = "pg-01"
  distributed_virtual_switch_uuid = vsphere_distributed_virtual_switch.vds.id

  vlan_id = 1000

  active_uplinks  = ["uplink1", "uplink2"]
  standby_uplinks = []
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the port group.
* `distributed_virtual_switch_uuid` - (Required) The ID of the VDS to add the
  port group to. Forces a new resource if changed.
* `type` - (Optional) The port group type. Can be one of `earlyBinding` (static
  binding) or `ephemeral`. Default: `earlyBinding`.
* `description` - (Optional) An optional description for the port group.
* `number_of_ports` - (Optional) The number of ports available on this port
  group. Cannot be decreased below the amount of used ports on the port group.
* `auto_expand` - (Optional) Allows the port group to create additional ports
  past the limit specified in `number_of_ports` if necessary. Default: `true`.

~> **NOTE:** Using `auto_expand` with a statically defined `number_of_ports`
may lead to errors when the port count grows past the amount specified.  If you
specify `number_of_ports`, you may wish to set `auto_expand` to `false`.

* `port_name_format` - (Optional) An optional formatting policy for naming of
  the ports in this port group. See the `portNameFormat` attribute listed
  [here][ext-vsphere-portname-format] for details on the format syntax.

[ext-vsphere-portname-format]: https://vdc-download.vmware.com/vmwb-repository/dcr-public/b50dcbbf-051d-4204-a3e7-e1b618c1e384/538cf2ec-b34f-4bae-a332-3820ef9e7773/vim.dvs.DistributedVirtualPortgroup.ConfigInfo.html#portNameFormat

* `network_resource_pool_key` - (Optional) The key of a network resource pool
  to associate with this port group. The default is `-1`, which implies no
  association.
* `custom_attributes` (Optional) Map of custom attribute ids to attribute
  value string to set for port group. See [here][docs-setting-custom-attributes] 
  for a reference on how to set values for custom attributes.

[docs-setting-custom-attributes]: /docs/providers/vsphere/r/custom_attribute.html#using-custom-attributes-in-a-supported-resource

~> **NOTE:** Custom attributes are not supported on direct ESXi host
connections and require vCenter Server.

### Policy options

In addition to the above options, you can configure any policy option that is
available under the `vsphere_distributed_virtual_switch` 
[policy options][vds-default-port-policies] section. Any policy option that is
not set is inherited from the vSphere Distributed Switch, its options
propagating to the port group.

See the link for a full list of options that can be set.

### Port override options

The following options below control whether or not the policies set in the port
group can be overridden on the individual port:

* `block_override_allowed` - (Optional) Allow the [port shutdown
  policy][port-shutdown-policy] to be overridden on an individual port.
* `live_port_moving_allowed` - (Optional) Allow a port in this port group to be
  moved to another port group while it is connected.
* `netflow_override_allowed` - (Optional) Allow the
  [Netflow policy][netflow-policy] on this port group to be overridden on an
  individual port.
* `network_resource_pool_override_allowed` - (Optional) Allow the network
  resource pool set on this port group to be overridden on an individual port.
* `port_config_reset_at_disconnect` - (Optional) Reset a port's settings to the
  settings defined on this port group policy when the port disconnects.
* `security_policy_override_allowed` - (Optional) Allow the 
  [security policy settings][sec-policy-settings] defined in this port group
  policy to be overridden on an individual port.
* `shaping_override_allowed` - (Optional) Allow the
  [traffic shaping options][traffic-shaping-settings] on this port group policy
  to be overridden on an individual port.
* `traffic_filter_override_allowed` - (Optional) Allow any traffic filters on
  this port group to be overridden on an individual port.
* `uplink_teaming_override_allowed` - (Optional) Allow the
  [uplink teaming options][uplink-teaming-settings] on this port group to be
  overridden on an individual port.
* `vlan_override_allowed` - (Optional) Allow the
  [VLAN settings][vlan-settings] on this port group to be overridden on an
  individual port.

[port-shutdown-policy]: /docs/providers/vsphere/r/distributed_virtual_switch.html#block_all_ports
[netflow-policy]: /docs/providers/vsphere/r/distributed_virtual_switch.html#netflow_enabled
[sec-policy-settings]: /docs/providers/vsphere/r/distributed_virtual_switch.html#security-options
[traffic-shaping-settings]: /docs/providers/vsphere/r/distributed_virtual_switch.html#traffic-shaping-options
[uplink-teaming-settings]: /docs/providers/vsphere/r/distributed_virtual_switch.html#ha-policy-options
[vlan-settings]: /docs/providers/vsphere/r/distributed_virtual_switch.html#vlan-options

## Attribute Reference

The following attributes are exported:

* `id`: The [managed object reference ID][docs-about-morefs] of the created
  port group.
* `key`: The generated UUID of the port group.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

~> **NOTE:** While `id` and `key` may look the same in state, they are
documented differently in the vSphere API and come from different fields in the
port group object. If you are asked to supply an
[managed object reference ID][docs-about-morefs] to another resource, be sure
to use the `id` field.

* `config_version`: The current version of the port group configuration,
  incremented by subsequent updates to the port group.

## Importing

An existing port group can be [imported][docs-import] into this resource using
the managed object id of the port group, via the following command:

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_distributed_port_group.pg  /dc-01/network/pg-01
```

The above would import the port group named `pg-01` that is located in the `dc-01`
datacenter.