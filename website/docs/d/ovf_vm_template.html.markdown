---
subcategory: "Virtual Machine"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_ovf_vm_template"
sidebar_current: "docs-vsphere-data-source-ovf-vm-template"
description: |-
A data source that can be used to extract the configuration of an OVF template 
---

# vsphere\_ovf\_vm\_template

The `vsphere_ovf_vm_template` data source can be used to submit an OVF to vSphere and extract its hardware
settings in a form that can be then used as inputs for a `vsphere_virtual_machine` resource.

## Example Usage

```hcl
data "vsphere_ovf_vm_template" "ovf" {
  name             = "testOVF"
  resource_pool_id = vsphere_resource_pool.rp.id
  datastore_id     = data.vsphere_datastore.ds.id
  host_system_id   = data.vsphere_host.hs.id
  remote_ovf_url   = "https://download3.vmware.com/software/vmw-tools/nested-esxi/Nested_ESXi7.0_Appliance_Template_v1.ova"

  ovf_network_map = {
    "Network 1": data.vsphere_network.net.id
  }
}
```

## Argument Reference

The following arguments are supported:
* `name` - Name of the virtual machine to create.
* `resource_pool_id` - (Required) The ID of a resource pool to put the virtual machine in.
* `host_system_id` - (Required) The ID of an optional host system to pin the virtual machine to.
* `datastore_id` - (Required) The ID of the virtual machine's datastore. The virtual machine configuration is placed here, along with any virtual disks that are created without datastores.
* `folder` - (Required) The name of the folder to locate the virtual machine in.
* `local_ovf_path` - (Optional) The absolute path to the ovf/ova file in the local system. While deploying from ovf,
  make sure the other necessary files like the .vmdk files are also in the same directory as the given ovf file.
* `remote_ovf_url` - (Optional) URL to the remote ovf/ova file to be deployed.

~> **NOTE:** Either `local_ovf_path` or `remote_ovf_url` is required, both can't be empty.

* `ip_allocation_policy` - (Optional) The IP allocation policy.
* `ip_protocol` - (Optional) The IP protocol.
* `disk_provisioning` - (Optional) The disk provisioning. If set, all the disks in the deployed OVF will have
  the same specified disk type (accepted values {thin, flat, thick, sameAsSource}).
* `deployment_option` - (Optional) The key of the chosen deployment option. If empty, the default option is chosen.
* `ovf_network_map` - (Optional) The mapping of name of network identifiers from the ovf descriptor to network UUID in the
  VI infrastructure.
* `allow_unverified_ssl_cert` - (Optional) Allow unverified ssl certificates while deploying ovf/ova from url.


## Attribute Reference
* `num_cpus` - The number of virtual processors to assign to this virtual machine.
* `num_cores_per_socket` - The number of cores to distribute amongst the CPUs in this virtual machine.
* `cpu_hot_add_enabled` - Allow CPUs to be added to this virtual machine while it is running.
* `cpu_hot_remove_enabled` - Allow CPUs to be added to this virtual machine while it is running.
* `nested_hv_enabled` - Enable nested hardware virtualization on this virtual machine, facilitating nested virtualization in the guest.
* `memory` - The size of the virtual machine's memory, in MB.
* `memory_hot_add_enabled` - Allow memory to be added to this virtual machine while it is running.
* `swap_placement_policy` - The swap file placement policy for this virtual machine.
* `annotation` - User-provided description of the virtual machine.
* `guest_id` - The guest ID for the operating system
* `alternate_guest_name` - The guest name for the operating system .
* `firmware` - The firmware interface to use on the virtual machine.

