---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_distributed_virtual_switch"
sidebar_current: "docs-vsphere-resource-distributed-virtual-switch"
description: |-
  Provides a VMware vSphere distributed virtual switch resource. A distributed switch configures a switch across all associated hosts, allowing a virtual machine to keep the same network configuration regardless of the actual host it runs in. 

  A distributed virtual switch has an uplink port group as well as several distributed port groups. The uplink port group connects physical network cards on the host to the distributed switch. A distributed port group specifies how a connection is made through the distributed switch.

---

# vsphere\_distributed_virtual_switch

Provides a VMware vSphere distributed virtual switch resource. A distributed switch configures a switch across all associated hosts, allowing a virtual machine to keep the same network configuration regardless of the actual host it runs in. 

## Example Usages

**Create a distributed virtual switch without specifying uplink port groups (need to be defined manually later):**

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "myDC"
}

resource "vsphere_distributed_virtual_switch" "myDistributedSwitch" {
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
  name = "myDistributedSwitch"
}
```

**Create a distributed virtual switch with connected hosts and their NICs as part of the uplink port group:**

```hcl
data "vsphere_datacenter" "datacenter" {
  name = "myDC"
}

data "vsphere_host" "esxi_host1" {
  name = "node1"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

data "vsphere_host" "esxi_host2" {
  name = "node2"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_distributed_virtual_switch" "myDistributedSwitch" {
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
  name   = "myDistributedSwitch"

  host = [{
    host_system_id = "${data.vsphere_host.esxi_host1.id}"
    backing = ["vmnic1","vmnic2"]
  },{
    host_system_id = "${data.vsphere_host.esxi_host2.id}"
    backing = ["vmnic0"]
  }]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the distributed virtual switch.
* `description` - (Optional) The description string of the distributed virtual switch.
* `default_proxy_switch_max_num_ports` - (Optional) The default host proxy switch maximum port number.
* `extension_key` - (Optional) The key of the extension registered by a remote server that controls the switch.
* `network_resource_control_version` - (Optional) Indicates the Network Resource Control APIs that are supported on the switch.
* `num_standalone_ports` - (Optional) The number of standalone ports in the switch. Standalone ports are ports that do not belong to any portgroup.
* `switch_ip_address` - (Optional) IP address for the switch, specified using IPv4 dot notation. IPv6 address is not supported for this property.
* `datacenter_id` - (Required) The ID of the datacenter where the distributed virtual switch will be created.
* `host` - (Optional) A list of hosts and physical NICs to attach to the uplink port group.
  * max_proxy_switch_ports - (Optional) Maximum number of ports allowed in the HostProxySwitch.
  * host_system_id - (Required) The managed object ID of the host to search for NICs on.
  * backing - (Optional) 
  * vendor_specific_config - (Optional) A list of key/blob vendor specific config. 
    * key - (Optional) A key that identifies the opaque binary blob.
    * opaque_data - (Optional) The opaque data. It is recommended that base64 encoding be used for binary data.
* `contact` - (Optional) The contact information for the person.
* `contact_name` - (Optional) The name of the person who is responsible for the switch.
