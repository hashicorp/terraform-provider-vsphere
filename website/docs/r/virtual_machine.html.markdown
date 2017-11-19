---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_virtual_machine"
sidebar_current: "docs-vsphere-resource-vm-virtual-machine"
description: |-
  Provides a VMware vSphere virtual machine resource. This can be used to create, modify, and delete virtual machines.
---

# vsphere\_virtual\_machine

The `vsphere_virtual_machine` resource can be used to manage the complex
lifecycle of a virtual machine. It supports management of disk, network
interface, and CDROM devices, creation from scratch or cloning from template,
and migration through both host and storage vMotion.

For more details on working with virtual machines in vSphere, see [this
page][vmware-docs-vm-management].

[vmware-docs-vm-management]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.vm_admin.doc/GUID-55238059-912E-411F-A0E9-A7A536972A91.html

## About Working with Virtual Machines in Terraform

A high degree of control and flexibility is afforded to a vSphere user when it
comes to how to configure, deploy, and manage virtual machines - much more
control than given in a traditional cloud provider. As such, Terraform has to
make some decisions on how to manage the virtual machines it creates and
manages. This section documents things you need to know about your virtual
machine configuration that you should consider when setting up virtual
machines, creating templates to clone from, or importing external virtual
machines into Terraform.

### Disks

The `vsphere_virtual_machine` resource currently only supports standard
VMDK-backed virtual disks - it does not support other special kinds of disk
devices like RDM disks.

Disks are managed by exact name supplied to the `name` attribute of a [`disk`
sub-resource](#disk-options). This is required - the resource does not support
automatic naming.

Virtual disks can be SCSI disks only. The entire SCSI bus is filled with
controllers of the type defined by the top-level [`scsi_type`](#scsi_type)
setting. If you are cloning from a template, devices will be added or
re-configured as necessary.

When cloning from a template, you must specify disks of either the same or
greater size than the disks in source template when creating a traditional
clone, or exactly the same size when cloning from snapshot (also known as a
linked clone). For more details, see the section on [creating a virtual machine
from a template](#creating-a-virtual-machine-from-a-template).

A maximum of 60 virtual disks can be configured. See the [disk
options](#disk-options) section for more details.

### Customization and network waiters

Terraform waits during various parts of a virtual machine deployment to ensure
that it is in a correct expected state before proceeding. These happen when a
VM is created, or also when it's updated, depending on the waiter.

Two waiters of note are:

* **The customization waiter:** This waiter watches events in vSphere to
  monitor when customization on a virtual machine completes during VM creation.
  Depending on your vSphere or VM configuration it may be necessary to change
  the timeout or turn it off. This can be controlled by the
  [`timeout`](#timeout-1) setting in the [customization
  settings](#virtual-machine-customization) block.
* **The network waiter:** This waiter waits for a _routeable_ interface to show
  up on a guest virtual machine close to the end of both VM creation and
  update. This waiter is necessary to ensure that correct IP information gets
  reported to the guest virtual machine, mainly to facilitate the availability
  of a valid, routeable default IP address for any
  [provisioners][tf-docs-provisioners]. This option can be managed or turned
  off via the [`wait_for_guest_net_timeout`](#wait_for_guest_net_timeout)
  top-level setting.

[tf-docs-provisioners]: http://localhost:4567/docs/provisioners/index.html

### Migrating from a previous version of this resource

~> **NOTE:** This section only applies to versions of this resource available
in versions v0.4.2 of this provider or earlier.

The path for migrating to the current version of this resource is very similar
to the [import](#importing) path, with the exception that the `terraform
import` command does not need to be run. See that section for details on what
is required before you run `terraform plan` on a state that requires migration.

## Example Usage

### Creating a virtual machine from scratch

The following block contains all that is necessary to create a new virtual
machine, with a single disk and network interface. 

The resource makes use of the following data sources to do its job:
[`vsphere_datacenter`][tf-vsphere-datacenter] to locate the datacenter,
[`vsphere_datastore`][tf-vsphere-datastore] to locate the default datastore to
put the virtual machine in, [`vsphere_resource_pool`][tf-vsphere-resource-pool]
to locate a resource pool located in a cluster or standalone host, and
[`vsphere_network`][tf-vsphere-network] to locate a network.

[tf-vsphere-datacenter]: /docs/providers/vsphere/d/datacenter.html
[tf-vsphere-datastore]: /docs/providers/vsphere/d/datastore.html
[tf-vsphere-resource-pool]: /docs/providers/vsphere/d/resource_pool.html
[tf-vsphere-network]: /docs/providers/vsphere/d/network.html

```hcl
data "vsphere_datacenter" "dc" {
  name = "dc1"
}

data "vsphere_datastore" "datastore" {
  name          = "datastore1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "cluster1/Resources"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "public"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 1024
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = 20
  }
}
```

### Cloning and customization example

Building on the above example, the below configuration creates a VM by cloning
it from a template, fetched via the
[`vsphere_virtual_machine`][tf-vsphere-virtual-machine-ds] data source. This
allows us to locate the UUID of the template we want to clone, along with
settings for network interface type, SCSI bus type (especially important on
Windows machines), and disk sizes.

[tf-vsphere-virtual-machine-ds]: /docs/providers/vsphere/d/virtual_machine.html

```hcl
data "vsphere_datacenter" "dc" {
  name = "dc1"
}

data "vsphere_datastore" "datastore" {
  name          = "datastore1"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "cluster1/Resources"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "public"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "ubuntu-16.04"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 1024
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"
  
  scsi_type = "${data.vsphere_virtual_machine.template.scsi_type}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = "${data.vsphere_virtual_machine.template.disk_sizes[0]}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "10.0.0.10"
        ipv4_netmask = 24
      }

      ipv4_gateway = "10.0.0.1"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

### General options

The following options are general virtual machine and Terraform workflow
options:

* `name` - (Required) The name of the virtual machine.
* `resource_pool_id` - (Required) The ID resource pool to put this virtual
  machine in. See the section on [virtual machine
  migration](#virtual-machine-migration) for details on changing this value.
* `datastore_id` - (Required) The ID of the virtual machine's datastore. The
  virtual machine configuration is placed here, along with any virtual disks
  that are created where a datastore is not explicitly specified. See the
  section on [virtual machine migration](#virtual-machine-migration) for
  details on changing this value.
* `folder` - (Optional) The path to the folder to put this virtual machine in,
  relative to the datacenter that the resource pool is in.
* `host_system_id` - (Optional) An optional ID of a host to put this virtual
  machine on. See the section on [virtual machine
  migration](#virtual-machine-migration) for details on changing this value. If
  a `host_system_id` is not supplied, vSphere will select a host in the
  resource pool to place the virtual machine, according to any defaults or DRS
  policies in place. 
* `disk` - (Required) A specification for a virtual disk device on this virtual
  machine. See [disk options](#disk-options) below.
* `network_interface` - (Required) A specification for a virtual NIC on this
  virtual machine. See [network interface options](#network-interface-options)
  below.
* `cdrom` - (Optional) A specification for a CDROM device on this virtual
  machine. See [CDROM options](#cdrom-options) below.
* `clone` - (Optional) When specified, the VM will be created as a clone of a
  specified template. Optional customization options can be submitted as well.
  See [creating a virtual machine from a
  template](#creating-a-virtual-machine-from-a-template) for more details.
* `guest_id` - (Optional) The guest ID for the operating system type. For a
  full list of possible values, see [here][vmware-docs-guest-ids]. Default: `other-64`.

[vmware-docs-guest-ids]: https://pubs.vmware.com/vsphere-6-5/topic/com.vmware.wssdk.apiref.doc/vim.vm.GuestOsDescriptor.GuestOsIdentifier.html

* `alternate_guest_name` - (Optional) The guest name for the operating system
  when `guest_id` is `other` or `other-64`.
* `enable_disk_uuid` - (Optional) Expose the UUIDs of attached virtual disks to
  the virtual machine, allowing access to them in the guest. Default: `false`.
* `hv_mode` - (Optional) The (non-nested) hardware virtualization setting for
  this virtual machine. Can be one of `hvAuto`, `hvOn`, or `hvOff`. Default:
  `hvAuto`.
* `ept_rvi_mode` - (Optional) The EPT/RVI (hardware memory virtualization)
  setting for this virtual machine. Can be one of `automatic`, `on`, or `off`.
  Default: `automatic`.
* `nested_hv_enabled` - (Optional) Enable nested hardware virtualization on
  this virtual machine, facilitating nested virtualization in the guest.
  Default: `false`.
* `enable_logging` - (Optional) Enable logging of virtual machine events to a
  log file stored in the virtual machine directory. Default: `true`.
* `cpu_performance_counters_enabled` - (Optional) Enable CPU performance
  counters on this virtual machine. Default: `false`.
* `swap_placement_policy` - (Optional) The swap file placement policy for this
  virtual machine. Can be one of `inherit`, `hostLocal`, or `vmDirectory`.
  Default: `inherit`.
* `annotation` - (Optional) A user-provided description of the virtual machine.
  The default is no annotation.
 `firmware` - (Optional) The firmware interface to use on the virtual machine.
  Can be one of `bios` or `EFI`. Default: `bios`.
* `extra_config` - (Optional) Extra configuration data for this virtual
  machine. Can be used to supply advanced parameters not normally in
  configuration, such as data for cloud-config (under the guestinfo namespace),
  or configuration data for OVF images.
* `wait_for_guest_net_timeout` - (Optional) The amount of time, in minutes, to
  wait for a routeable IP address on this virtual machine. A value less than 1
  disables the waiter. Defualt: 5 minutes.
* `shutdown_wait_timeout` - (Optional) The amount of time, in minutes, to wait
  for a graceful guest shutdown when making necessary updates to the virtual
  machine. If `force_power_off` is set to true, the VM will be force powered-off
  after this timeout, otherwise an error is returned. Default: 3 minutes.
* `migrate_wait_timeout` - (Optional) The amount of time, in minutes, to wait
  for a virtual machine migration to complete before failing. Default: 10
  minutes. Also see the section on [virtual machine
  migration](#virtual-machine-migration).
* `force_power_off` - (Optional) If a guest shutdown failed or timed out while
  updating or destroying (see
  [`shutdown_wait_timeout`](#shutdown_wait_timeout)), force the power-off of
  the virtual machine. Default: `true`.
* `scsi_type` - (Optional) The type of SCSI bus this virtual machine will have.
  Can be one of lsilogic (LSI Logic Parallel), lsilogic-sas (LSI Logic SAS) or
  pvscsi (VMware Paravirtual). Defualt: `lsilogic`.
* `tags` - (Optional) The IDs of any tags to attach to this resource. See
  [here][docs-applying-tags] for a reference on how to apply tags.

[docs-applying-tags]: /docs/providers/vsphere/r/tag.html#using-tags-in-a-supported-resource

~> **NOTE:** Tagging support is unsupported on direct ESXi connections and
requires vCenter 6.0 or higher.

### CPU and memory options

The following options control CPU and memory settings on the virtual machine:

* `num_cpus` - (Optional) The number of virtual processors to assign to this
  virtual machine. Default: `1`.
* `num_cores_per_socket` - (Optional) The number of cores to distribute among
  the CPUs in this virtual machine. If specified, the value supplied to
  `num_cpus` must be evenly divisible by this value. Default: `1`.
* `cpu_hot_add_enabled` - (Optional) Allow CPUs to be added to this virtual
  machine while it is running.
* `cpu_hot_remove_enabled` - (Optional) Allow CPUs to be removed to this
  virtual machine while it is running.
* `memory` - (Optional) The size of the virtual machine's memory, in MB.
  Default: `1024` (1 GB).
* `memory_hot_add_enabled` - (Optional) Allow memory to be added to this
  virtual machine while it is running.

~> **NOTE:** Certain CPU and memory hot-plug options are not available on every
operating system. Check the [VMware Guest OS Compatibility
Guide][vmware-docs-compat-guide] first to see what settings your guest
operating system is eligible for. In addition, at least one `terraform apply`
must be executed before being able to take advantage of CPU and memory hot-plug
settings, so if you want the support, enable it as soon as possible.

[vmware-docs-compat-guide]: http://partnerweb.vmware.com/comp_guide2/pdf/VMware_GOS_Compatibility_Guide.pdf

### Boot options 

The following options control boot settings on the virtual machine:

* `boot_delay` - (Optional) The number of milliseconds to wait before starting
  the boot sequence. The default is no delay.
* `efi_secure_boot_enabled` - (Optional) When the `firmware` type is set to is
  `efi`, this enables EFI secure boot. Default: `false`.
* `boot_retry_delay` - (Optional) The number of milliseconds to wait before
  retrying the boot sequence. This only valid if `boot_retry_enabled` is true.
  Default: `10000` (10 seconds).
* `boot_retry_enabled` - (Optional) If set to true, a virtual machine that
  fails to boot will try again after the delay defined in `boot_retry_delay`.
  Default: `false`.

### VMware Tools options

The following options control VMware tools options on the virtual machine:

* `sync_time_with_host` - (Optional) Enable guest clock synchronization with
  the host. Requires VMware tools to be installed. Default: `false`.
* `run_tools_scripts_after_power_on` - (Optional) Enable the execution of
  post-power-on scripts when VMware tools is installed. Default: `true`.
* `run_tools_scripts_after_resume` - (Optional) Enable the execution of
  post-resume scripts when VMware tools is installed. Default: `true`.
* `run_tools_scripts_before_guest_reboot` - (Optional) Enable the execution of
  pre-reboot scripts when VMware tools is installed. Default: `false`.
* `run_tools_scripts_before_guest_shutdown` - (Optional) Enable the execution
  of pre-shutdown scripts when VMware tools is installed. Default: `true`.
* `run_tools_scripts_before_guest_standby` - (Optional) Enable the execution of
  pre-standby scripts when VMware tools is installed. Default: `true`.

### Resource allocation options

The following options allow control over CPU and memory allocation on the
virtual machine. Note that the resource pool that this VM is in may affect
these options.

* `cpu_limit` - (Optional) The maximum amount of CPU (in MHz) that this virtual
  machine can consume, regardless of available resources. The default is no
  limit.
* `cpu_reservation` - (Optional) The amount of CPU (in MHz) that this virtual
  machine is guaranteed. The default is no reservation.
* `cpu_share_level` - (Optional) The allocation level for CPU resources. Can be
  one of `high`, `low`, `normal`, or `custom`. Default: `custom`.
* `cpu_share_count` - (Optional) The number of CPU shares allocated to the
  virtual machine when the `cpu_share_level` is `custom`.
* `memory_limit` - (Optional) The maximum amount of memory (in MB) that this
  virtual machine can consume, regardless of available resources. The default
  is no limit.
* `memory_reservation` - (Optional) The amount of memory (in MB) that this
  virtual machine is guaranteed. The default is no reservation.
* `memory_share_level` - (Optional) The allocation level for memory resources.
  Can be one of `high`, `low`, `normal`, or `custom`. Default: `custom`.
* `memory_share_count` - (Optional) The number of memory shares allocated to
  the virtual machine when the `memory_share_level` is `custom`.


### Disk options

Virtual disks are managed by adding an instance of the `disk` sub-resource.

At the very least, there must be `name` and `size` attributes. `unit_number` is
required for any disk other than the first, and there must be at least one
resource with the implicit number of 0.

An abridged multi-disk example is below:

```hcl
resource "vsphere_virtual_machine" "vm" {
  ...

  disk {
    name = "terraform-test.vmdk"
    size = "10"
  }
  
  disk {
    name        = "terraform-test_data.vmdk"
    size        = "100"
    unit_number = 1
  }

  ...
}
```

The options are:

* `name` - (Required) The name of the disk. Forces a new disk if changed.  This
  should only be a longer path (example: `directory/disk.vmdk`) if attaching an
  external disk.
* `size` - (Required) The size of the disk, in GiB.
* `unit_number` - (Optional) The disk number on the SCSI bus. Can be one of `0`
  to `59`. The default is `0`, for which one disk must be set to. Duplicate
  unit numbers are not allowed.
* `datastore_id` - (Optional) The datastore for this virtual disk. The default
  is to use the datastore of the virtual machine. See the section on [virtual
  machine migration](#virtual-machine-migration) for details on changing this
  value.
* `attach` - (Optional) Attach an external disk instead of creating a new one.
  Implies and conflicts with `keep_on_remove`. If set, you cannot set `size`,
  `eagerly_scrub`, or `thin_provisioned`.
* `keep_on_remove` - (Optional) Keep this disk when removing the sub-resource
  or destroying the virtual machine. Default: `false`.
* `disk_mode` - (Optional) The mode of this this virtual disk for purposes of
  writes and snapshotting. Can be one of `append`, `independent_nonpersistent`,
  `independent_persistent`, `nonpersistent`, `persistent`, or `undoable`.
  Default: `persistent`. For an explanation of options, click
  [here][vmware-docs-disk-mode].

[vmware-docs-disk-mode]: https://pubs.vmware.com/vsphere-6-5/topic/com.vmware.wssdk.apiref.doc/vim.vm.device.VirtualDiskOption.DiskMode.html

* `eagerly_scrub` - (Optional) If set to `true`, the disk space is zeroed out
  on VM creation. This will delay the creation of the disk or virtual machine.
  See the section on [picking a disk type](#picking-a-disk-type).  Default:
  `false`.
* `thin_provisioned` - (Optional) If `true`, this disk is thin provisioned, with
  space for the file being allocated on an as-needed basis. See the section on
  [picking a disk type](#picking-a-disk-type). Default: `true`. 
* `disk_sharing` - (Optional) The sharing mode of this virtual disk. Can be one
  of `sharingMultiWriter` or `sharingNone`. Default: `sharingNone`.
* `write_through` - (Optional) If `true`, writes for this disk are sent
  directly to the filesystem immediately instead of being buffered. Default:
  `false`.
* `io_limit` - (Optional) The upper limit of IOPS that this disk can use. The
  default is no limit.
* `io_reservation` - (Optional) The I/O reservation (guarantee) that this disk
  has, in IOPS.  The default is no reservation.
* `io_share_level` - (Optional) The share allocation level for this disk. Can
  be one of `low`, `normal`, `high`, or `custom`. Default: `normal`.
* `io_share_count` - (Optional) The share count for this disk when the share
  level is `custom`.

#### Picking a disk type

The `eagerly_scrub` and `thin_provisioned` options control the space allocation
type of a virtual disk. These show up in the vSphere console as a unified
enumeration of options, the equivalents of which are explained below. The
defaults in the sub-resource are the equivalent of thin provisioning.

* **Thick provisioned lazy zeroed:** Both `eagerly_scrub` and
  `thin_provisioned` should be set to `false`.
* **Thick provisioned eager zeroed:** `eagerly_scrub` should be set to true,
  and `thin_provisioned` should be set to `false`.
* **Thin provisioned:** `eagerly_scrub` should be set to `false`, and
  `thin_provisioned` should be set to `true`.

~> **NOTE:** Not all disk types are available on some types of datastores.
Attempting to set the appropriate options on disks on these datastores will
create spurious diffs in Terraform.

~> **NOTE:** The disk type cannot be changed once set.

#### Reviewing a disk diff

Due to the nature of the options in the `disk` sub-resource, each instance
needs to be assigned a unique hash in the terraform state versus simply being
assigned a sequence number, unlike `network_interface` and `cdrom` devices.
Hence, when running `terraform plan` after making changes to a disk, you will
see a diff similar to the output below:

```
disk.1657266812.attach:           "" => "false"
disk.1657266812.datastore_id:     "" => "datastore-123"
disk.1657266812.device_address:   "" => "scsi:0:0"
disk.1657266812.disk_mode:        "" => "persistent"
disk.1657266812.disk_sharing:     "" => "sharingNone"
disk.1657266812.eagerly_scrub:    "" => "false"
disk.1657266812.io_limit:         "" => "-1"
disk.1657266812.io_reservation:   "" => "0"
disk.1657266812.io_share_count:   "" => "1000"
disk.1657266812.io_share_level:   "" => "normal"
disk.1657266812.keep_on_remove:   "" => "false"
disk.1657266812.key:              "" => "2000"
disk.1657266812.name:             "" => "terraform-test/terraform-test.vmdk"
disk.1657266812.size:             "" => "50"
disk.1657266812.thin_provisioned: "" => "true"
disk.1657266812.unit_number:      "" => "0"
disk.1657266812.write_through:    "" => "false"
disk.2182090100.attach:           "false" => "false"
disk.2182090100.datastore_id:     "datastore-123" => ""
disk.2182090100.disk_mode:        "persistent" => ""
disk.2182090100.disk_sharing:     "sharingNone" => ""
disk.2182090100.eagerly_scrub:    "false" => "false"
disk.2182090100.io_limit:         "-1" => "0"
disk.2182090100.io_reservation:   "0" => "0"
disk.2182090100.io_share_count:   "1000" => "0"
disk.2182090100.io_share_level:   "normal" => ""
disk.2182090100.keep_on_remove:   "false" => "false"
disk.2182090100.name:             "terraform-test/terraform-test.vmdk" => ""
disk.2182090100.size:             "32" => "0"
disk.2182090100.thin_provisioned: "true" => "false"
disk.2182090100.unit_number:      "0" => "0"
disk.2182090100.write_through:    "false" => "false"
```

To make sure you have a functional diff:

* Locate the hashes on that match the disk's `name` field on the left and
  right. These are your old and new sets for each affected disk. Disks that are
  not changing (not shown here) will have data on both the left and right and
  should not show up in the diff with a different hash number.
* Check to make sure that `key` is not `0` and `device_address` is not showing
  up on the right as `<computed>`. If they are, review your configuration, make
  any necessary changes and re-run `terraform plan`. **Do not** apply the plan
  as it was generated as you may lose data on the affected disk.
* You should now be able to compare the rest of the fields in the diff. Here,
  we are increasing the `size` of the disk from 32 to 50 GiB.

### Network interface options

Network interfaces are managed by adding an instance of the `network_interface`
sub-resource.

Interfaces are assigned to devices in the specific order they are declared.
This has different implications for different operating systems.

Given the following example:

```hcl
resource "vsphere_virtual_machine" "vm" {
  ...

  network_interface {
    network_id   = "${data.vsphere_network.public.id}"
  }

  network_interface {
    network_id   = "${data.vsphere_network.private.id}"
  }
}
```

The first interface with the `public` network assigned to it would show up in
order before the interface assigned to `private`. On some Linux systems, this
might mean that the first interface would show up as `eth0` and the second
would show up as `eth1`.

The options are:

* `network_id` - (Required) The ID of the network to connect this interface
  to.
* `adapter_type` - (Optional) The network interface type. Can be one of
  `e1000`, `e1000e`, or `vmxnet3`. Default: `e1000`.
* `use_static_mac` - (Optional) If true, the `mac_address` field is treated as
  a static MAC address and set accordingly. Setting this to `true` requires
  `mac_address` to be set. Default: `false`.
* `mac_address` - (Optional) The MAC address of this network interface. Can
  only be manually set if `use_static_mac` is true, otherwise this is a
  computed value that gives the current MAC address of this interface.
* `bandwidth_limit` - (Optional) The upper bandwidth limit of this network
  interface, in Mbits/sec. The default is no limit.
* `bandwidth_reservation` - (Optional) The bandwidth reservation of this
  network interface, in Mbits/sec. The default is no reservation.
* `bandwidth_share_level` - (Optional) The bandwidth share allocation level for
  this interface. Can be one of `low`, `normal`, `high`, or `custom`. Default:
  `normal`.
* `bandwidth_share_count` - (Optional) The share count for this network
  interface when the share level is `custom`.

### CDROM options

A single virtual CDROM device can be created and attached to the virtual
machine. The resource only supports attaching a CDROM from a datastore ISO.

An example is below:

```hcl
resource "vsphere_virtual_machine" "vm" {
  ...

  cdrom {
    datastore_id = "${data.vsphere_datastore.iso_datastore.id}"
    path         = "ISOs/os-livecd.iso"
  }
}
```

The options are:

* `datastore_id` - (Required) The datastore ID that the ISO is located in.
* `path` - (Required) The path to the ISO file.

### Virtual device computed options

Virtual device resources (`disk`, `network_interface`, and `cdrom`) all export
the following attributes. These options help locate the sub-resource on future
Terraform runs. The options are:

* `key` - The ID of the device within the virtual machine.
* `device_address` - An address internal to Terraform that helps locate the
  device when `key` is unavailable. This follows a convention of
  `CONTROLLER_TYPE:BUS_NUMBER:UNIT_NUMBER`. Example: `scsi:0:1` means device
  unit 1 on SCSI bus 0.

## Creating a Virtual Machine from a Template

The `clone` sub-resource can be used to create a new virtual machine from an
existing virtual machine or template. The resource supports both making a
complete copy of a virtual machine, or cloning from a snapshot (otherwise known
as a linked clone).

For see the [cloning and customization
example](#cloning-and-customization-example) for a usage synopsis.

~> **NOTE:** Changing any option in `clone` after creation forces a new
resource.

The options available in the `clone` sub-resource are:

* `template_uuid` - (Required) The UUID of the source virtual machine or
  template.
* `linked_clone` - (Optional) Clone this virtual machine from a snapshot.
  Templates must have a single snapshot only in order to be eligible. Default:
  `false`.
* `timeout` - (Optional) The timeout, in minutes, to wait for the virtual
  machine clone to complete. Default: 10 minutes.
* `customize` - (Optional) The customization spec for this clone. This allows
  the user to configure the virtual machine post-clone. For more details, see
  [virtual machine customization](#virtual-machine-customization).

### Virtual machine customization

As part of the `clone` operation, a virtual machine can be
[customized][vmware-docs-customize] to configure host, network, or licensing
settings.

[vmware-docs-customize]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.vm_admin.doc/GUID-58E346FF-83AE-42B8-BE58-253641D257BC.html

To perform virtual machine customization as a part of the clone process,
specify the `customize` sub-resource within the `clone` sub-resource with the
respective customization options.  See the [cloning and customization
example](#cloning-and-customization-example) for a usage synopsis.

The settings for `customize` are as follows:

#### Customization timeout settings

* `timeout` - (Optional) The time, in minutes that Terraform waits for
  customization to complete before failing. The default is 10 minutes, and
  setting the value to 0 or a negative value disables the waiter altogether.

#### Network interface settings

The following settings should be in a `network_interface` block in the
`customize` sub-resource. These settings configure network interfaces on a
per-interface basis and are matched up to `network_interface` sub-resources in
the main block in the order they are declared.

Given the following example:

```hcl
resource "vsphere_virtual_machine" "vm" {
  ...

  network_interface {
    network_id   = "${data.vsphere_network.public.id}"
  }

  network_interface {
    network_id   = "${data.vsphere_network.private.id}"
  }

  clone {
    ...

    customize {
      ...

      network_interface {
        ipv4_address = "10.0.0.10"
        ipv4_netmask = 24
      }
      
      network_interface {
        ipv4_address = "172.16.0.10"
        ipv4_netmask = 24
      }

      ipv4_gateway = "10.0.0.1"
    }
  }
}
```

The first set of `network_interface` data would be assigned to the `public`
interface, and the second to the `private` interface.

The options are:

* `dns_server_list` - (Optional) Network interface-specific DNS server settings
  for Windows operating systems. Ignored on Linux and possibly other opearating
  systems - for those systems, please see the [global DNS
  settings](#global-dns-settings) section.
* `dns_server_list` - (Optional) Network interface-specific DNS search domain
  for Windows operating systems. Ignored on Linux and possibly other opearating
  systems - for those systems, please see the [global DNS
  settings](#global-dns-settings) section.
* `ipv4_address` - (Optional) The IPv4 address assigned to this network adapter. If left
  blank or not included, DHCP is used.
* `ipv4_netmask` The IPv4 subnet mask, in bits (example: `24` for
  255.255.255.0).
* `ipv6_address` - (Optional) The IPv6 address assigned to this network adapter. If left
  blank or not included, auto-configuration is used.
* `ipv6_netmask` - (Optional) The IPv6 subnet mask, in bits (example: `32`).

#### Global routing settings

VM customization under the `vsphere_virtual_machine` resource does not take a
per-interface gateway setting, but rather default routes are configured on a
global basis. For an example, see the [network interface settings
section](#network-interface-settings).

The settings here must match the IP/mask of at least one `network_interface`
supplied to customization.

The options are:

* `ipv4_gateway` - (Optional) The IPv4 default gateway when using
  `network_interface` customization on the virtual machine.
* `ipv6_gateway` - (Optional) The IPv6 default gateway when using
  `network_interface` customization on the virtual machine.

#### Global DNS settings

The following settings configure DNS globally, generally for Linux systems. For
Windows systems, this is done per-interface, see [network interface
settings](#network-interface-settings).

* `dns_server_list` - The list of DNS servers to configure on a virtual
  machine. 
* `dns_suffix_list` - A list of DNS search domains to add to the DNS
  configuration on the virtual machine.

#### Linux customization options

The settings in the `linux_options` sub-resource pertain to Linux guest OS
customization. If you are customizing a Linux operating system, this section
must be included.

Example:

```hcl
resource "vsphere_virtual_machine" "vm" {
  ...

  clone {
    ...

    customize {
      ...

      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }
    }
  }
}
```

The options are:

* `host_name` - (Required) The host name for this machine. This, along with
  `domain`, make up the FQDN of this virtual machine.
* `domain` - (Required) The domain name for this machine. This, along with
  `host_name`, make up the FQDN of this virtual machine.
* `hw_clock_utc` - (Optional) Tells the operating system that the hardware
  clock is set to UTC. Default: `true`.
* `time_zone` - (Optional) Sets the time zone. For a list of possible
  combinations, click [here][vmware-docs-valid-linux-tzs]. The default is UTC.

[vmware-docs-valid-linux-tzs]: https://pubs.vmware.com/vsphere-6-5/topic/com.vmware.wssdk.apiref.doc/timezone.html

#### Windows customization options

The settings in the `windows_options` sub-resource pertain to Windows guest OS
customization. If you are customizing a Windows operating system, this section
must be included.

Example:

```hcl
resource "vsphere_virtual_machine" "vm" {
  ...

  clone {
    ...

    customize {
      ...

      windows_options {
        computer_name  = "terraform-test"
        workgroup      = "test"
        admin_password = "VMw4re"
      }
    }
  }
}
```

The options are:

* `computer_name` - (Required) The computer name of this virtual machine.
* `admin_password` - (Optional) The administrator password for this virtual
  machine.

~> **NOTE:** `admin_password` is a sensitive field in Terraform and will not be
output on-screen, but is stored in state and sent to the VM in plain text -
keep this in mind when provisioning your infrastructure.

* `workgroup` - (Optional) The workgroup name for this virtual machine. One of
  this or `join_domain` must be included.
* `join_domain` - (Optional) The domain to join for this virtual machine. One
  of this or `workgroup` must be included.
* `domain_admin_user` - (Optional) The user of the domain administrator used to
  join this virtual machine to the domain. Required if you are setting `join_domain`.
* `domain_admin_password` - (Optional) The password of the domain administrator
  used to join this virtual machine to the domain. Required if you are setting
  `join_domain`.

~> **NOTE:** `domain_admin_password` is a sensitive field in Terraform and will
not be output on-screen, but is stored in state and sent to the VM in plain
text - keep this in mind when provisioning your infrastructure.

* `full_name` - (Optional) The full name of the user of this virtual machine.
  This populates the "user" field in the general Windows system information.
  Default: `Administrator`.
* `organization_name` - (Optional) The organization name this virtual machine
  is being installed for.  This populates the "organization" field in the
  general Windows system information.  Default: `Managed by Terraform`.
* `product_key` - (Optional) The product key for this virtual machine. The
  default is no key.
* `run_once_command_list` - (Optional) A list of commands to run at first user
  logon, after guest customization.
* `auto_logon` - (Optional) Specifies whether or not the VM automatically logs
  on as Administrator. Default: `false`.
* `auto_logon_count` - (Optional) Specifies how many times the VM should auto-logon
  the Administrator account when `auto_logon` is true. This should be set
  accordingly to ensure that all of your commands that run in
  `run_once_command_list` can log in to run. Default: `1`.
* `time_zone` - (Optional) The new time zone for the virtual machine. This is a
  numeric, sysprep-dictated, timezone code. For a list of codes, click
  [here][ms-docs-valid-sysprep-tzs]. The default is `85` (GMT/UTC).

[ms-docs-valid-sysprep-tzs]: https://msdn.microsoft.com/en-us/library/ms912391(v=winembedded.11).aspx

#### Supplying your own SysPrep file

Alternative to the `windows_options` supplied above, you can instead supply
your own `sysprep.inf` file contents via the `windows_sysprep_text` option.
This allows full control of the customization process out-of-band of vSphere.
Example below:

```hcl
resource "vsphere_virtual_machine" "vm" {
  ...

  clone {
    ...

    customize {
      ...

      windows_sysprep_text = "${file("${path.module}/sysprep.inf")}"
    }
  }
}
```

Note this option is mutually exclusive to `windows_options` - one must not be
included if the other is specified.

### Additional requirements and notes for cloning

Note that when cloning from a template, there are additional requirements in
both the resource configuration and source template:

* All disks on the virtual machine must be SCSI disks.
* You must specify at least the same number of `disk` sub-resources as there
  are disks that exist in the template. These sub-resources are ordered and
  lined up by the `unit_number` attribute. Additional disks can be added past
  this.
* The `size` of a virtual disk must be at least the same size as its
  counterpart disk in the template.
* When using `linked_clone`, the `size` has to be an exact match.
* Some operating systems (such as Windows) do not respond well to a change in
  disk controller type, so when using such OSes, take care to ensure that
  `scsi_type` is set to an exact match of the template's controller set. For
  maximum compatibility, make sure the SCSI controllers on the source template
  are all the same type.

To ease the gathering of some of these options, you can use the
[`vsphere_virtual_machine` data source][tf-vsphere-virtual-machine-ds], which
will give you disk sizes, network interface types, SCSI bus types, and also the
guest ID of the source template.  See the [cloning and customization
example](#cloning-and-customization-example) for usage details.

## Virtual Machine Migration

The `vsphere_virtual_machine` resource supports live migration (otherwise known
as vMotion) both on the host and storage level. One can migrate the entire VM
to another host or datastore, and migrate or pin a single disk to a specific
datastore.

### Host migration 

To migrate the virtual machine to another host or resource pool, change the
`host_system_id` or `resource_pool_id` to the manged object IDs of the new host
or resource pool accordingly.

The same rules apply for migration as they do for VM creation - any host
specified needs to be a part of the resource pool supplied.

### Storage migration

Storage migration can be done on two levels:

* Global datastore migration can be handled by changing the global
  `datastore_id` attribute. This triggers a storage migration for all disks
  that do not have an explicit `datastore_id` specified.
* An individual `disk` sub-resource can be migrated by manually specifying the
  `datastore_id` in its sub-resource. This also pins it to the specific
  datastore that is specified - if at a later time the VM and any unpinned
  disks migrate to another host, the disk will stay on the specified datastore.

An example of datastore pinning is below. As long as the datastore in the
`pinned_datastore` data source does not change, any change to the standard
`vm_datastore` data source will not affect the data disk - the disk will stay
where it is.

```hcl
resource "vsphere_virtual_machine" "vm" {
  ...

  datastore_id     = "${data.vsphere_datastore.vm_datastore.id}"

  disk {
    name = "terraform-test.vmdk"
    size = 10
  }
  
  disk {
    datastore_id = "${data.vsphere_datastore.pinned_datastore.id}"
    name         = "terraform-test_data.vmdk"
    size         = 100
    unit_number  = 1
  }

  ...
}
```

## Attribute Reference

The following attributes are exported on the base level of this resource:

* `id` - The UUID of the virtual machine.
* `reboot_required` - Value internal to Terraform used to determine if a
  configuration set change requires a reboot. This value is only useful during
  an update process and gets reset on refresh.
* `vmware_tools_status` - The state of VMware tools in the guest. This will
  determine the proper course of action for some device operations.
* `vmx_path` - The path of the virtual machine's configuration file in the VM's
  datastore.
* `imported` - This is flagged if the virtual machine has been imported, or the
  state has been migrated from a previous version of the resource, and blocks
  the `clone` configuration option from being set. See the section on
  [importing](#importing) below.
* `change_version` - A unique identifier for a given version of the last
  configuration applied, such the timestamp of the last update to the
  configuration.
* `uuid` - The UUID of the virtual machine. Also exposed as the `id` of the
  resource.

## Importing 

An existing virtual machine can be [imported][docs-import] into this resource via
the full path to the virtual machine, via the following command:

[docs-import]: /docs/import/index.html

```
terraform import vsphere_virtual_machine.vm /dc1/vm/srv1
```

The above would import the virtual machine named `srv1` that is located in the
`dc1` datacenter.

### Additional requirements and notes for importing

Many of the same requirements for
[cloning](#additional-requirements-and-notes-for-cloning) apply to importing,
although since importing writes directly to state, a lot of these rules cannot
be enforced at import time, so every effort should be made to ensure the
correctness of the configuration before the import.

In addition to these rules, the following extra rules apply to importing:

* Disks need to be named the exact same as they appear in the VM configuration.
  It is recommended that you have no snapshots on the virtual machine before
  reviewing the names of the disks in configuration, as snapshots create delta
  disks that obfuscate the names of the parent.
* Disks always get imported with [`keep_on_remove`](#keep_on_remove) enabled
  until the first `terraform apply` runs, which will remove the setting for
  known disks. This is an extra safeguard against naming or accounting mistakes
  in the disk configuration.
* You cannot use the [`clone`](#clone) sub-resource on any imported VM. If you
  need to clone a new virtual machine or want a working configuration with
  `clone` features, you will need to create a new resource and destroy the old
  one.

After importing, you should run `terraform plan` and review the plan. It will
always be non-empty due to several TF-specific options that need to be added to
state, in addition to finalizing the configuration for virtual disks that get
added, and normalization of he SCSI bus. Some of these changes require a power
off of the virtual machine, so plan accordingly. Please see the section on
[reviewing a disk diff](#reviewing-a-disk-diff) for information on how to read
the disk diff details. Unless you have changed anything else in configuration
that would be causing disk attributes to change, the only difference should be
the transition of [`keep_on_remove`](#keep_on_remove) of known disks from
`true` to `false`.
