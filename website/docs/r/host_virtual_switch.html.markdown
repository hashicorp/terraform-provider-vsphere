---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_host_virtual_switch"
sidebar_current: "docs-vsphere-resource-networking-host-virtual-switch"
description: |-
  Provides a vSphere Host Virtual Switch Resource. This can be used to configure vSwitches direct on an ESXi host.
---

# vsphere\_host\_virtual\_switch

The `vsphere_host_virtual_switch` resource can be used to manage vSphere
standard switches on an ESXi host. These switches can be used as a backing for
standard port groups, which can be managed by the
[`vsphere_host_port_group`][host-port-group] resource.

For an overview on vSphere networking concepts, see [this
page][ref-vsphere-net-concepts].

[host-port-group]: /docs/providers/vsphere/r/host_port_group.html
[ref-vsphere-net-concepts]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.networking.doc/GUID-2B11DBB8-CB3C-4AFF-8885-EFEA0FC562F4.html

## Example Usages

**Create a virtual switch with one active and one standby NIC:**

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc1"
}

data "vsphere_host" "host" {
  name          = "esxi1"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_host_virtual_switch" "switch" {
  name           = "vSwitchTerraformTest"
  host_system_id = "${data.vsphere_host.host.id}"

  network_adapters = ["vmnic0", "vmnic1"]

  active_nics  = ["vmnic0"]
  standby_nics = ["vmnic1"]
}
```

**Create a virtual switch with extra networking policy options:**

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "dc1"
}

data "vsphere_host" "host" {
  name          = "esxi1"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_host_virtual_switch" "switch" {
  name           = "vSwitchTerraformTest"
  host_system_id = "${data.vsphere_host.host.id}"

  network_adapters = ["vmnic0", "vmnic1"]

  active_nics    = ["vmnic0"]
  standby_nics   = ["vmnic1"]
  teaming_policy = "failover_explicit"

  allow_promiscuous      = false
  allow_forged_transmits = false
  allow_mac_changes      = false

  shaping_enabled           = true
  shaping_average_bandwidth = 50000000
  shaping_peak_bandwidth    = 100000000
  shaping_burst_size        = 1000000000
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the virtual switch. Forces a new resource if
  changed.
* `host_system_id` - (Required) The [managed object ID][docs-about-morefs] of
  the host to set the virtual switch up on. Forces a new resource if changed.
* `mtu` - (Optional) The maximum transmission unit (MTU) for the virtual
  switch. Default: `1500`.
* `number_of_ports` - (Optional) The number of ports to create with this
  virtual switch. Default: `128`.

[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

~> **NOTE:** Changing the port count requires a reboot of the host. Terraform
will not restart the host for you.

### Bridge Options

The following arguments are related to how the virtual switch binds to physical
NICs:

* `network_adapters` - (Required) The network interfaces to bind to the bridge.
* `beacon_interval` - (Optional) The interval, in seconds, that a NIC beacon
  packet is sent out. This can be used with [`check_beacon`](#check_beacon) to
  offer link failure capability beyond link status only. Default: `1`.
* `link_discovery_operation` - (Optional) Whether to `advertise` or `listen`
  for link discovery traffic. Default: `listen`.
* `link_discovery_protocol` - (Optional) The discovery protocol type.  Valid
  types are `cpd` and `lldp`. Default: `cdp`.

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

* `active_nics` - (Required) The list of active network adapters used for load
  balancing.
* `standby_nics` - (Required) The list of standby network adapters used for
  failover.
* `check_beacon` - (Optional) Enable beacon probing - this requires that the
  [`beacon_interval`](#beacon_interval) option has been set in the bridge
  options. If this is set to `false`, only link status is used to check for
  failed NICs.  Default: `false`.
* `teaming_policy` - (Optional) The network adapter teaming policy. Can be one
  of `loadbalance_ip`, `loadbalance_srcmac`, `loadbalance_srcid`, or
  `failover_explicit`. Default: `loadbalance_srcid`.
* `notify_switches` - (Optional) If set to `true`, the teaming policy will
  notify the broadcast network of a NIC failover, triggering cache updates.
  Default: `true`.
* `failback` - (Optional) If set to `true`, the teaming policy will re-activate
  failed interfaces higher in precedence when they come back up.  Default:
  `true`.

#### Security Policy Options

* `allow_promiscuous` - (Optional) Enable promiscuous mode on the network. This
  flag indicates whether or not all traffic is seen on a given port. Default:
  `false`.
* `allow_forged_transmits` - (Optional) Controls whether or not the virtual
  network adapter is allowed to send network traffic with a different MAC
  address than that of its own. Default: `true`.
* `allow_mac_changes` - (Optional) Controls whether or not the Media Access
  Control (MAC) address can be changed. Default: `true`.

#### Traffic Shaping Options

* `shaping_enabled` - (Optional) Set to `true` to enable the traffic shaper for
  ports managed by this virtual switch. Default: `false`.
* `shaping_average_bandwidth` - (Optional) The average bandwidth in bits per
  second if traffic shaping is enabled. Default: `0`
* `shaping_peak_bandwidth` - (Optional) The peak bandwidth during bursts in
  bits per second if traffic shaping is enabled. Default: `0`
* `shaping_burst_size` - (Optional) The maximum burst size allowed in bytes if
  shaping is enabled. Default: `0`

## Attribute Reference

The only exported attribute, other than the attributes above, is the `id` of
the resource. This is set to an ID value unique to Terraform - the convention
is a prefix, the host system ID, and the virtual switch name. An example would
be `tf-HostVirtualSwitch:host-10:vSwitchTerraformTest`.

## Importing

An existing vSwitch can be [imported][docs-import] into this resource by its ID.
The convention of the id is a prefix, the host system [managed objectID][docs-about-morefs], and the virtual switch
name. An example would be `tf-HostVirtualSwitch:host-10:vSwitchTerraformTest`.
Import can the be done via the following command:

[docs-import]: https://www.terraform.io/docs/import/index.html
[docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

```
terraform import vsphere_host_virtual_switch.switch tf-HostVirtualSwitch:host-10:vSwitchTerraformTest
```

The above would import the vSwtich named `vSwitchTerraformTest` that is located in the `host-10`
vSphere host.
