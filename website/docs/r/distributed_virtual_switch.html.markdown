---
subcategory: "Networking"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_distributed_virtual_switch"
sidebar_current: "docs-vsphere-resource-networking-distributed-virtual-switch"
description: |-
  Provides a vSphere distributed virtual switch resource. This can be used to create and manage DVS resources in vCenter.
---

# vsphere\_distributed\_virtual\_switch

The `vsphere_distributed_virtual_switch` resource can be used to manage VMware
Distributed Virtual Switches.

An essential component of a distributed, scalable VMware datacenter, the
vSphere Distributed Virtual Switch (DVS) provides centralized management and
monitoring of the networking configuration of all the hosts that are associated
with the switch. In addition to adding port groups (see the
[`vsphere_distributed_port_group`][distributed-port-group] resource) that can
be used as networks for virtual machines, a DVS can be configured to perform
advanced high availability, traffic shaping, network monitoring, and more.

For an overview on vSphere networking concepts, see [this
page][ref-vsphere-net-concepts]. For more information on vSphere DVS, see [this
page][ref-vsphere-dvs].

[distributed-port-group]: /docs/providers/vsphere/r/distributed_port_group.html
[ref-vsphere-net-concepts]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.networking.doc/GUID-2B11DBB8-CB3C-4AFF-8885-EFEA0FC562F4.html
[ref-vsphere-dvs]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.networking.doc/GUID-375B45C7-684C-4C51-BA3C-70E48DFABF04.html

~> **NOTE:** This resource requires vCenter and is not available on direct ESXi
connections.

## Example Usage

The following example below demonstrates a "standard" example of configuring a
vSphere DVS in a 3-node vSphere datacenter named `dc1`, across 4 NICs with two
being used as active, and two being used as passive. Note that the NIC failover
order propagates to any port groups configured on this DVS and can be overridden
there.

```hcl
variable "esxi_hosts" {
  default = [
    "esxi1",
    "esxi2",
    "esxi3",
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

data "vsphere_datacenter" "dc" {
  name = "dc1"
}

data "vsphere_host" "host" {
  count         = "${length(var.esxi_hosts)}"
  name          = "${var.esxi_hosts[count.index]}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_distributed_virtual_switch" "dvs" {
  name          = "terraform-test-dvs"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"

  uplinks         = ["uplink1", "uplink2", "uplink3", "uplink4"]
  active_uplinks  = ["uplink1", "uplink2"]
  standby_uplinks = ["uplink3", "uplink4"]

  host {
    host_system_id = "${data.vsphere_host.host.0.id}"
    devices        = ["${var.network_interfaces}"]
  }

  host {
    host_system_id = "${data.vsphere_host.host.1.id}"
    devices        = ["${var.network_interfaces}"]
  }

  host {
    host_system_id = "${data.vsphere_host.host.2.id}"
    devices        = ["${var.network_interfaces}"]
  }
}
```

### Uplink name and count control

The following abridged example below demonstrates how you can manage the number
of uplinks, and the name of the uplinks via the `uplinks` parameter.

Note that if you change the uplink naming and count after creating the DVS, you
may need to explicitly specify `active_uplinks` and `standby_uplinks` as these
values are saved to Terraform state after creation, regardless of being
specified in config, and will drift if not modified, causing errors.

```hcl
resource "vsphere_distributed_virtual_switch" "dvs" {
  name          = "terraform-test-dvs"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"

  uplinks         = ["tfup1", "tfup2"]
  active_uplinks  = ["tfup1"]
  standby_uplinks = ["tfup2"]
}
```

~> **NOTE:** The default uplink names when a DVS is created are `uplink1`
through to `uplink4`, however this default is not guaranteed to be stable and
you are encouraged to set your own.

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the distributed virtual switch.
* `datacenter_id` - (Required) The ID of the datacenter where the distributed
  virtual switch will be created. Forces a new resource if changed.
* `folder` - (Optional) The folder to create the DVS in. Forces a new resource
  if changed.
* `description` - (Optional) A detailed description for the DVS.
* `contact_name` - (Optional) The name of the person who is responsible for the
  DVS. 
* `contact_detail` - (Optional) The detailed contact information for the person
  who is responsible for the DVS. 
* `ipv4_address` - (Optional) An IPv4 address to identify the switch. This is
  mostly useful when used with the [Netflow arguments](#netflow-arguments) found
  below.
* `lacp_api_version` - (Optional) The Link Aggregation Control Protocol group
  version to use with the switch. Possible values are `singleLag` and
  `multipleLag`.
* `link_discovery_operation` - (Optional) Whether to `advertise` or `listen`
  for link discovery traffic.
* `link_discovery_protocol` - (Optional) The discovery protocol type. Valid
  types are `cdp` and `lldp`.
* `max_mtu` - (Optional) The maximum transmission unit (MTU) for the virtual
  switch.
* `multicast_filtering_mode` - (Optional) The multicast filtering mode to use
  with the switch. Can be one of `legacyFiltering` or `snooping`.
* `version` - (Optional) - The version of the DVS to create. The default is to
  create the DVS at the latest version supported by the version of vSphere
  being used. A DVS can be upgraded to another version, but cannot be
  downgraded.
* `tags` - (Optional) The IDs of any tags to attach to this resource. See
  [here][docs-applying-tags] for a reference on how to apply tags.

[docs-applying-tags]: /docs/providers/vsphere/r/tag.html#using-tags-in-a-supported-resource

~> **NOTE:** Tagging support requires vCenter 6.0 or higher.

* `custom_attributes` - (Optional) Map of custom attribute ids to attribute
  value strings to set for virtual switch. See 
  [here][docs-setting-custom-attributes] for a reference on how to set values 
  for custom attributes.

[docs-setting-custom-attributes]: /docs/providers/vsphere/r/custom_attribute.html#using-custom-attributes-in-a-supported-resource

~> **NOTE:** Custom attributes are unsupported on direct ESXi connections 
and require vCenter.

### Uplink arguments

* `uplinks` - (Optional) A list of strings that uniquely identifies the names
  of the uplinks on the DVS across hosts. The number of items in this list
  controls the number of uplinks that exist on the DVS, in addition to the
  names.  See [here](#uplink-name-and-count-control) for an example on how to
  use this option.

### Host management arguments

* `host` - (Optional) Use the `host` block to declare a host specification. The
  options are:
 * `host_system_id` - (Required) The host system ID of the host to add to the
   DVS.
 * `devices` - (Required) The list of NIC devices to map to uplinks on the DVS,
   added in order they are specified.

### Netflow arguments

The following options control settings that you can use to configure Netflow on
the DVS:

* `netflow_active_flow_timeout` - (Optional) The number of seconds after which
  active flows are forced to be exported to the collector. Allowed range is
  `60` to `3600`. Default: `60`. 
* `netflow_collector_ip_address` - (Optional) IP address for the Netflow
  collector, using IPv4 or IPv6. IPv6 is supported in vSphere Distributed
  Switch Version 6.0 or later. Must be set before Netflow can be enabled.
* `netflow_collector_port` - (Optional) Port for the Netflow collector. This
  must be set before Netflow can be enabled.
* `netflow_idle_flow_timeout` - (Optional) The number of seconds after which
  idle flows are forced to be exported to the collector. Allowed range is `10`
  to `600`. Default: `15`.
* `netflow_internal_flows_only` - (Optional) Whether to limit analysis to
  traffic that has both source and destination served by the same host.
  Default: `false`.
* `netflow_observation_domain_id` - (Optional) The observation domain ID for
  the Netflow collector.
* `netflow_sampling_rate` - (Optional) The ratio of total number of packets to
  the number of packets analyzed. The default is `0`, which indicates that the
  switch should analyze all packets. The maximum value is `1000`, which
  indicates an analysis rate of 0.001%.

### Network I/O control arguments

The following arguments manage network I/O control. Network I/O control (also
known as network resource control) can be used to set up advanced traffic
shaping for the DVS, allowing control of various classes of traffic in a
fashion similar to how resource pools work for virtual machines. Configuration
of network I/O control is also a requirement for the use of network resource
pools, if their use is so desired.

#### General network I/O control arguments

* `network_resource_control_enabled` - (Optional) Set to `true` to enable
  network I/O control. Default: `false`.
* `network_resource_control_version` - (Optional) The version of network I/O
  control to use. Can be one of `version2` or `version3`. Default: `version2`.

#### Network I/O control traffic classes

There are currently 9 traffic classes that can be used for network I/O
control - they are below.

Each of these classes has 4 options that can be tuned that are discussed in the
next section.

<table>
<tr><th>Type</th><th>Class Name</th></tr>
<tr><td>Fault Tolerance (FT) Traffic</td><td>`faulttolerance`</td></tr>
<tr><td>vSphere Replication (VR) Traffic</td><td>`hbr`</td></tr>
<tr><td>iSCSI Traffic</td><td>`iscsi`</td></tr>
<tr><td>Management Traffic</td><td>`management`</td></tr>
<tr><td>NFS Traffic</td><td>`nfs`</td></tr>
<tr><td>vSphere Data Protection</td><td>`vdp`</td></tr>
<tr><td>Virtual Machine Traffic</td><td>`virtualmachine`</td></tr>
<tr><td>vMotion Traffic</td><td>`vmotion`</td></tr>
<tr><td>VSAN Traffic</td><td>`vsan`</td></tr>
</table>

#### Traffic class resource options

There are 4 traffic resource options for each class, prefixed with the name of
the traffic classes seen above.

For example, to set the traffic class resource options for virtual machine
traffic, see the example below:

```hcl
resource "vsphere_distributed_virtual_switch" "dvs" {
  # ... other configuration ...
  virtualmachine_share_level      = "custom"
  virtualmachine_share_count      = 150
  virtualmachine_maximum_mbit     = 200
  virtualmachine_reservation_mbit = 20
}
```

The options are:

* `share_level` - (Optional) A pre-defined share level that can be assigned to
  this resource class. Can be one of `low`, `normal`, `high`, or `custom`.
* `share_count` - (Optional) The number of shares for a custom level. This is
  ignored if `share_level` is not `custom`.
* `maximum_mbit` - (Optional) The maximum amount of bandwidth allowed for this
  traffic class in Mbits/sec.
* `reservation_mbit` - (Optional) The guaranteed amount of bandwidth for this
  traffic class in Mbits/sec.

### Default port group policy arguments

The following arguments are shared with the
[`vsphere_distributed_port_group`][distributed-port-group] resource. Setting
them here defines a default policy here that will be inherited by other port
groups on this switch that do not have these values otherwise overridden. Not
defining these options in a DVS will infer defaults that can be seen in the
Terraform state after the initial apply.

Of particular note to a DVS are the [HA policy options](#ha-policy-options),
which is where the `active_uplinks` and `standby_uplinks` options are
controlled, allowing the ability to create a NIC failover policy that applies
to the entire DVS and all portgroups within it that don't override the policy.

#### VLAN options

The following options control the VLAN behaviour of the port groups the port
policy applies to. One one of these 3 options may be set:

* `vlan` - (Optional) The member VLAN for the ports this policy applies to. A
  value of `0` means no VLAN.
* `vlan_range` - (Optional) Used to denote VLAN trunking. Use the `min_vlan`
  and `max_vlan` sub-arguments to define the tagged VLAN range. Multiple
  `vlan_range` definitions are allowed, but they must not overlap. Example
  below:

```hcl
resource "vsphere_distributed_virtual_switch" "dvs" {
  # ... other configuration ...
  vlan_range {
    min_vlan = 1
    max_vlan = 1000
  }
  vlan_range {
    min_vlan = 2000
    max_vlan = 4094
  }
}
```

* `port_private_secondary_vlan_id` - (Optional) Used to define a secondary VLAN
  ID when using private VLANs.

#### HA policy options

The following options control HA policy for ports that this policy applies to:

* `active_uplinks` - (Optional) A list of active uplinks to be used in load
  balancing. These uplinks need to match the definitions in the
  [`uplinks`](#uplinks) DVS argument. See
  [here](#uplink-name-and-count-control) for more details.
* `standby_uplinks` - (Optional) A list of standby uplinks to be used in
  failover. These uplinks need to match the definitions in the
  [`uplinks`](#uplinks) DVS argument. See
  [here](#uplink-name-and-count-control) for more details.
* `check_beacon` - (Optional) Enables beacon probing as an additional measure
  to detect NIC failure.

~> **NOTE:** VMware recommends using a minimum of 3 NICs when using beacon
probing.

* `failback` - (Optional) If `true`, the teaming policy will re-activate failed
  uplinks higher in precedence when they come back up.
* `notify_switches` - (Optional) If `true`, the teaming policy will notify the
  broadcast network of an uplink failover, triggering cache updates.
* `teaming_policy` - (Optional) The uplink teaming policy. Can be one of
  `loadbalance_ip`, `loadbalance_srcmac`, `loadbalance_srcid`, or
  `failover_explicit`.

#### LACP options

The following options allow the use of LACP for NIC teaming for ports that this
policy applies to.

~> **NOTE:** These options are ignored for non-uplink port groups and hence are
only useful at the DVS level.

* `lacp_enabled` - (Optional) Enables LACP for the ports that this policy
  applies to.
* `lacp_mode` - (Optional) The LACP mode. Can be one of `active` or `passive`.

#### Security options

The following options control security settings for the ports that this policy
applies to:

* `allow_forged_transmits` - (Optional) Controls whether or not a virtual
  network adapter is allowed to send network traffic with a different MAC
  address than that of its own.
* `allow_mac_changes` - (Optional) Controls whether or not the Media Access
  Control (MAC) address can be changed.
* `allow_promiscuous` - (Optional) Enable promiscuous mode on the network. This
  flag indicates whether or not all traffic is seen on a given port.

#### Traffic shaping options

The following options control traffic shaping settings for the ports that this
policy applies to:

* `ingress_shaping_enabled` - (Optional) `true` if the traffic shaper is
  enabled on the port for ingress traffic.
* `ingress_shaping_average_bandwidth` - (Optional) The average bandwidth in
  bits per second if ingress traffic shaping is enabled on the port.
* `ingress_shaping_peak_bandwidth` - (Optional) The peak bandwidth during
  bursts in bits per second if ingress traffic shaping is enabled on the port.
* `ingress_shaping_burst_size` - (Optional) The maximum burst size allowed in
  bytes if ingress traffic shaping is enabled on the port.
* `egress_shaping_enabled` - (Optional) `true` if the traffic shaper is enabled
  on the port for egress traffic.
* `egress_shaping_average_bandwidth` - (Optional) The average bandwidth in bits
  per second if egress traffic shaping is enabled on the port.
* `egress_shaping_peak_bandwidth` - (Optional) The peak bandwidth during bursts
  in bits per second if egress traffic shaping is enabled on the port.
* `egress_shaping_burst_size` - (Optional) The maximum burst size allowed in
  bytes if egress traffic shaping is enabled on the port.

#### Miscellaneous options

The following are some general options that also affect ports that this policy
applies to:

* `block_all_ports` - (Optional) Shuts down all ports in the port groups that
  this policy applies to, effectively blocking all network access to connected
  virtual devices.
* `netflow_enabled` - (Optional) Enables Netflow on all ports that this policy
  applies to.
* `tx_uplink` - (Optional) Forward all traffic transmitted by ports for which
  this policy applies to its DVS uplinks.
* `directpath_gen2_allowed` - (Optional) Allow VMDirectPath Gen2 for the ports
  for which this policy applies to.

## Attribute Reference

The following attributes are exported:

* `id`: The UUID of the created DVS.
* `config_version`: The current version of the DVS configuration, incremented
  by subsequent updates to the DVS.

## Importing

An existing DVS can be [imported][docs-import] into this resource via the path
to the DVS, via the following command:

[docs-import]: https://www.terraform.io/docs/import/index.html

```
terraform import vsphere_distributed_virtual_switch.dvs /dc1/network/dvs
```

The above would import the DVS named `dvs` that is located in the `dc1`
datacenter.
