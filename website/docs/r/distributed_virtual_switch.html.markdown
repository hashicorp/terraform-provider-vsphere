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
resource "vsphere_distributed_virtual_switch" "myDistributedSwitch" {
  datacenter = "myDC"
  name = "myDistributedSwitch"
}
```

**Create a distributed virtual switch specifying uplink port groups):**

```hcl
resource "vsphere_distributed_virtual_switch" "myDistributedSwitch" {
  datacenter = "myDC"
  name   = "myDistributedSwitch"
  uplinks = { "10.0.30.25" = "vmnic1", "host100.mydomain.net" = "vmnic1"  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the distributed virtual switch.
* `datacenter` - (Required) The name of the datacenter containing the distributed virtual switch.
* `uplinks` - (Optional) A map of hosts and physical NICs to attach to the uplink port group.

~> **NOTE**: Distributed virtual switches cannot be changed once they are created. Modifying any of these attributes will force a new resource!
