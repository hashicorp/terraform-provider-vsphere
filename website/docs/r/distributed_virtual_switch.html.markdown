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
* `datacenter_id` - (Required) The ID of the datacenter where the distributed virtual switch will be created.
* `host` - (Optional) A map of hosts and physical NICs to attach to the uplink port group.
