---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_host_virtual_switch"
sidebar_current: "docs-vsphere-resource-host-virtual-switch"
description: |-
  Provides a vSphere Host Virtual Switch Resource. This can be used to configure vSwitches direct on an ESXi host.
---

# vsphere\_host\_virtual\_switch

The `vsphere_host_virtual_switch` resource can be used to manage vSphere
standard switches on an ESXi host. These switches can be used as a backing for
standard port groups, which can be managed by the
[`vsphere_host_port_group`][host-port-group] resource.

For an overview on vSphere networking concepts, see [this page][ref-vsphere-net-concepts].

[host-port-group]: /docs/providers/vsphere/r/host_port_group.html
[ref-vsphere-net-concepts]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.networking.doc/GUID-2B11DBB8-CB3C-4AFF-8885-EFEA0FC562F4.html

## Example Usages

**Create a virtual switch with one active and one standby NIC:**

```hcl
resource "vsphere_host_virtual_switch" "switch" {
  name             = "vSwitchTerraformTest"
  host             = "esx1.vsphere-lab.internal"
  datacenter       = "lab-dc1"
  network_adapters = ["vmnic0", "vmnic1"]

  active_nics  = ["vmnic0"]
  standby_nics = ["vmnic1"]
}
```

**Create a virtual switch with extra networking policy options:**

```hcl
resource "vsphere_host_virtual_switch" "switch" {
  name             = "vSwitchTerraformTest"
  host             = "esx1.vsphere-lab.internal"
  datacenter       = "lab-dc1"
  network_adapters = ["vmnic0", "vmnic1"]

  active_nics    = ["vmnic0"]
  standby_nics   = ["vmnic1"]
  teaming_policy = "failover_explicit"

  allow_promiscuous = false
  forged_transmits  = false
  mac_changes       = false

  shaping_enabled           = true
  shaping_average_bandwidth = 50000000
  shaping_peak_bandwidth    = 100000000
  shaping_burst_size        = 1000000000
}
```

## Argument Reference

The following arguments are supported:

* `name` - (String, required) The name of the virtual switch.
* `host` - (String) The host the virtual switch goes on. Required when using
  vCenter, not required when using ESXi.
* `datacenter` - (String) The name of the datacenter the host is in. Required
  when using vCenter, not required when using ESXi.
* `mtu` - (Integer, optional) The maximum transmission unit (MTU) for the virtual
  switch. Default: `1500`.
* `number_of_ports` - (Integer, optional) The number of ports to create with
  this virtual switch. Default: `128`.

~> **NOTE:** Changing the port count requires a reboot of the host. Terraform
will not restart the host for you.

### Bridge Options

The following arguments are related to how the virtual switch binds to physical
NICs:

* `network_adapters` - (Array of strings, required) The network interfaces to
  bind to the bridge.
* `beacon_interval` - (Integer, optional) The interval, in seconds, that a NIC
  beacon packet is sent out. This can be used with
  [`check_beacon`](#check_beacon) to offer link failure capability beyond link
  status only. Default: `1`.
* `link_discovery_operation` - (String, optional) Whether to `advertise` or
  `listen` for link discovery traffic. Default: `listen`.
* `link_discovery_protocol` - (String, optional) The discovery protocol type.
  Valid types are `cpd` and `lldp`. Default: `cdp`.

### Policy Options

The following options relate to how network traffic is handled on this virtual
switch. It also controls the NIC failover order. This subset of options is
shared with the [`vsphere_host_port_group`][host-port-group] resource, in which
options can be omitted to ensure options are inherited from the switch
configuration here.

#### NIC Teaming Options

~> **NOTE on NIC failover order:** An adapter can be in `active_nics`,
`standby_nics`, or neither to flag it as unused. However, virtual switch
creation or update operations will fail if a NIC is present in both settings,
or if the NIC is not a valid NIC in `network_adapters`.

~> **NOTE:** VMware recommends using a minimum of 3 NICs when using beacon
probing (configured with [`check_beacon`](#check_beacon)).

* `active_nics` - (Array of strings, required) The list of active network
  adapters used for load balancing.
* `standby_nics` - (Array of strings, required) The list of standby network
  adapters used for failover.
* `check_beacon` - (Boolean, optional) Enable beacon probing - this requires
  that the [`beacon_interval`](#beacon_interval) option has been set in the
  bridge options. If this is false, only link status is used to check for
  failed NICs. Default: `false`.
* `teaming_policy` - (String, optional) The network adapter teaming policy. Can
  be one of `loadbalance_ip`, `loadbalance_srcmac`, `loadbalance_srcid`, or
  `failover_explicit`. Default: `loadbalance_srcid`.
* `notify_switches` - (Boolean, optional) If `true`, the teaming policy will
  notify the broadcast network of a NIC failover, triggering cache updates.
  Default: `true`.
* `failback` - (Boolean, optional) If `true`, the teaming policy will
  re-activate failed interfaces higher in precedence when they come back up.
  Default: `true`.

#### Security Policy Options

* `allow_promiscuous` - (Boolean, optional) Enable promiscuous mode on the
  network. This flag indicates whether or not all traffic is seen on a given
  port. Default: `false`.
* `forged_transmits` - (Boolean, optional) Controls whether or not the virtual
  network adapter is allowed to send network traffic with a different MAC
  address than that of its own. Default: `true`.
* `mac_changes` - (Boolean, optional) Controls whether or not the Media Access
  Control (MAC) address can be changed. Default: `true`.

#### Traffic Shaping Options

* `shaping_enabled` - (Boolean, optional) `true` if the traffic shaper is
  enabled on the port. Default: `false`.
* `shaping_average_bandwidth` - (Integer, optional) The average bandwidth in
  bits per second if shaping is enabled on the port. Default: `0`
* `shaping_peak_bandwidth` - (Integer, optional) The peak bandwidth during
  bursts in bits per second if traffic shaping is enabled on the port. Default:
  `0`
* `shaping_burst_size` - (Integer, optional) The maximum burst size allowed in
  bytes if shaping is enabled on the port. Default: `0`

## Attribute Reference

The only exported attribute, other than the attributes above, is the `id` of
the resource, which is set to the name of the virtual switch.
