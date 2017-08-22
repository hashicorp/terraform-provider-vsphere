---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_host_port_group"
sidebar_current: "docs-vsphere-resource-host-port-group"
description: |-
  Provides a vSphere Host Port Group Resource. This can be used to configure port groups direct on an ESXi host.
---

# vsphere\_host\_port\_group

The `vsphere_host_port_group` resource can be used to manage vSphere standard
port groups on an ESXi host. These port groups are connected to standard
virtual switches, which can be managed by the
[`vsphere_host_virtual_switch`][host-virtual-switch] resource.

For an overview on vSphere networking concepts, see [this page][ref-vsphere-net-concepts].

[host-virtual-switch]: /docs/providers/vsphere/r/host_virtual_switch.html
[ref-vsphere-net-concepts]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.networking.doc/GUID-2B11DBB8-CB3C-4AFF-8885-EFEA0FC562F4.html

## Example Usages

**Create a virtual switch and bind a port group to it:**

```hcl
resource "vsphere_host_virtual_switch" "switch" {
  name             = "vSwitchTerraformTest"
  host             = "esx1.vsphere-lab.internal"
  datacenter       = "lab-dc1"
  network_adapters = ["vmnic0", "vmnic1"]

  active_nics  = ["vmnic0"]
  standby_nics = ["vmnic1"]
}

resource "vsphere_host_port_group" "pg" {
  name                = "PGTerraformTest"
  host                = "esx1.vsphere-lab.internal"
  datacenter          = "lab-dc1"
  virtual_switch_name = "${vsphere_host_virtual_switch.switch.id}"
}
```

**Create a port group with VLAN set and some overrides:**

This example sets the trunk mode VLAN (`4095`, which passes through all tags)
and sets
[`allow_promiscuous`](/docs/providers/vsphere/r/host_virtual_switch.html#allow_promiscuous)
to ensure that all traffic is seen on the port. The latter setting overrides
the implicit default of `false` set on the virtual switch.

```hcl
resource "vsphere_host_virtual_switch" "switch" {
  name             = "vSwitchTerraformTest"
  host             = "esx1.vsphere-lab.internal"
  datacenter       = "lab-dc1"
  network_adapters = ["vmnic0", "vmnic1"]

  active_nics  = ["vmnic0"]
  standby_nics = ["vmnic1"]
}

resource "vsphere_host_port_group" "pg" {
  name                = "PGTerraformTest"
  host                = "esx1.vsphere-lab.internal"
  datacenter          = "lab-dc1"
  virtual_switch_name = "${vsphere_host_virtual_switch.switch.id}"

  vlan_id = 4095

  allow_promiscuous = true
}
```

## Argument Reference

The following arguments are supported:

* `name` - (String, required, forces new resource) The name of the port group.
* `host` - (String, forces new resource) The host the port group goes on.
  Required when using vCenter, not required when using ESXi.
* `datacenter` - (String, forces new resource) The name of the datacenter the
  host is in. Required when using vCenter, not required when using ESXi.
* `virtual_switch_name` - (String, required, forces new resource) The name of
  the virtual switch to bind this port group to.
* `vlan_id` - (Integer, optional) The VLAN ID/trunk mode for this port group.
  An ID of `0` denotes no tagging, an ID of `1`-`4094` tags with the specific ID, and
  an ID of `4095` enables trunk mode, allowing the guest to manage its own
  tagging. Default: `0`.

### Policy Options

In addition to the above options, you can configure any policy option that is
available under the [`vsphere_host_virtual_switch` policy options
section][host-vswitch-policy-options].  Any policy option that is not set is
**inherited** from the virtual switch, its options propagating to the port
group.

See the link for a full list of options that can be set.

[host-vswitch-policy-options]: /docs/providers/vsphere/r/host_virtual_switch.html#policy-options

## Attribute Reference

The following attributes are exported:

* `id` - The name of the port group.
* `comptued_policy` - A map with a full set of the [policy
  options][host-vswitch-policy-options] computed from defaults and overrides,
  explaining the effective policy for this port group.
* `key` - The key for this port group as returned from the vSphere API.
* `ports` - A list of ports that currently exist and are used on this port group.
