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
sub-resource](#disk-subresource-options). This is required - Terraform does not
support automatic naming.

Virtual disks can be SCSI disks only. The entire SCSI bus is filled with
controllers of the type defined by the top-level `scsi_type` setting. If you
are cloning from a template, devices will be added or re-configured as
necessary.

When cloning from a template, you must specify disks of either the same or
greater size than the disks in source template when creating a traditional
clone, or exactly the same size when cloning from snapshot (also known as a
linked clone). For more details, see [cloning](#cloning).

A maximum of 60 virtual disks can be configured.

### Customization and network waiters

Terraform waits during various parts of a virtual machine deployment to ensure
that it is in a correct expected state before proceeding. These happen when a
VM is created, or also when it's updated, depending on the waiter.

Two waiters of note are:

* **The customization waiter:** This waiter watches events in vSphere to
  monitor when customization on a virtual machine completes during VM creation.
  Depending on your vSphere or VM configuration it may be necessary to change
  the time out or turn it off. This can be controlled by the `timeout` setting
  in the [customization settings](#customization-settings) block.
* **The network waiter:** This waiter waits for a _routeable_ interface to show
  up on a guest virtual machine close to the end of both VM creation and
  update. This waiter is necessary to ensure that correct IP information gets
  reported to the guest virtual machine, mainly to facilitate the availability
  of a valid, routeable default IP address for any
  [provisioners][tf-docs-provisioners]. This option can be managed or turned
  off via the [`wait_for_guest_net_timeout`](#wait_for_guest_net_timeout)
  top-level setting.

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

* `name` - (Required) The name of the virtual machine.
* `resource_pool_id` - (Required) The ID resource pool to put this virtual
  machine in. Also see the section on [virtual machine
  migration](virtual-machine-migration) for details on changing this value.
* `datastore_id` - (Required) The ID of the virtual machine's datastore. The
  virtual machine configuration is placed here, along with any virtual disks
  that are created where a datastore is not explicitly specified.
* `folder` - (Optional) The path to the folder to put this virtual machine in,
  relative to the datacenter that the resource pool is in.
* `host_system_id` - (Optional) An optional ID of a host to put this virtual
  machine on. Also see the section on [virtual machine
  migration](virtual-machine-migration) for details on changing this value. If
  a `host_system_id` is not supplied, vSphere will select a host in the
  resource pool to place the virtual machine, according to any defaults or DRS
  policies in place. 
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
* `enable_disk_uuid` - (Optional) Expose the UUIDs of attached virtual disks to
  the virtual machine, allowing access to them in the guest. Default: `true`.
* `hv_mode` - (Optional) The (non-nested) hardware virtualization setting for
  this virtual machine. Can be one of `hvAuto`, `hvOn`, or `hvOff`. Default:
  `hvAuto`.
* `ept_rvi_mode` - (Optional) The EPT/RVI (hardware memory virtualization)
  setting for this virtual machine. Can be one of `automatic`, `on`, or `off`.
  Default: `automatic`.
* `enable_logging` - (Optional) Enable logging of virtual machine events to a
  log file stored in the virtual machine directory. Default: `true`.
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
* `num_cpus` - (Optional) The number of virtual processors to assign to this
  virtual machine. Default: `1`.
* `num_cores_per_socket` - (Optional) The number of cores to distribute among
  the CPUs in this virtual machine. If specified, the value supplied to
  `num_cpus` must be evenly divisible by this value. Default: `1`.
* `cpu_hot_add_enabled` - (Optional) Allow CPUs to be added to this virtual
  machine while it is running. See the section on [live virtual machine
  modification](#live-virtual-machine-modification).
* `cpu_hot_remove_enabled` - (Optional) Allow CPUs to be removed to this
  virtual machine while it is running. See the section on [live virtual machine
  modification](#live-virtual-machine-modification).
* `memory` - (Optional) The size of the virtual machine's memory, in MB.
  Default: `1024` (1 GB).
* `memory_hot_add_enabled` - (Optional) Allow memory to be added to this
  virtual machine while it is running. See the section on [live virtual machine
  modification](#live-virtual-machine-modification).
* `nested_hv_enabled` - (Optional) Enable nested hardware virtualization on
  this virtual machine, facilitating nested virtualization in the guest.
  Default: `false`.
* `cpu_performance_counters_enabled` - (Optional) Enable CPU performance
  counters on this virtual machine. Default: `false`.
* `swap_placement_policy` - (Optional) The swap file placement policy for this
  virtual machine. Can be one of `inherit`, `hostLocal`, or `vmDirectory`.
  Default: `inherit`.
* `annotation` - (Optional) A user-provided description of the virtual machine.
  The default is no annotation.
* `guest_id` - (Optional) The guest ID for the operating system type. For a
  full list of possible values, see [here][vmware-docs-guest-ids]. Default: `other-64`.

[vmware-docs-guest-ids]: https://pubs.vmware.com/vsphere-6-5/topic/com.vmware.wssdk.apiref.doc/vim.vm.GuestOsDescriptor.GuestOsIdentifier.html

* `alternate_guest_name` - (Optional) The guest name for the operating system
  when `guest_id` is `other` or `other-64`.
* `firmware` - (Optional) The firmware interface to use on the virtual machine.
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
  migration](virtual-machine-migration).
* `force_power_off` - (Optional) If a guest shutdown failed or timed out while
  updating or destroying (see
  [`shutdown_wait_timeout`](#shutdown_wait_timeout)), force the power-off of
  the virtual machine. Default: `true`.
* `scsi_type` - (Optional) The type of SCSI bus this virtual machine will have.
  Can be one of lsilogic (LSI Logic Parallel), lsilogic-sas (LSI Logic SAS) or
  pvscsi (VMware Paravirtual). Defualt: `lsilogic`.
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
* `tags` - (Optional) The IDs of any tags to attach to this resource. See
  [here][docs-applying-tags] for a reference on how to apply tags.

[docs-applying-tags]: /docs/providers/vsphere/r/tag.html#using-tags-in-a-supported-resource

~> **NOTE:** Tagging support is unsupported on direct ESXi connections and
requires vCenter 6.0 or higher.

### Disk options

Virtual disks are managed by adding an instance of the `disk` sub-resource.

At the very least, there must be `name` and `size` attributes. `unit_number` is
required for any disk other than the first, and there must be at least one
resource with the implicit number of 0.

An abridged multi-disk example is below:

```hcl
resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  ...

  disk {
    name = "terraform-test.vmdk"
    size = "10"
  }
  
  disk {
    name        = "terraform-test_1.vmdk"
    size        = "100"
    unit_number = 1
  }

  ...
}
```

The options are:

* `name` - (Required) The name of the disk. Forces a new resource if changed.
  This should only be a longer path (example: `directory/disk.vmdk`) if
  attaching an external disk.
* `size` - (Required) The size of the disk, in GiB.
* `unit_number` - (Optional) The disk number on the SCSI bus. Can be one of `0`
  to `59`. The default is `0`, for which one disk must be set to. Duplicate
  unit numbers are not allowed.
* `datastore_id` - (Optional) The datastore for this virtual disk. The default
  is to use the datastore of the virtual machine. Also see the section on
  [virtual machine migration](virtual-machine-migration) for details on
  changing this value.
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
* `thin_provisioned` - (Optional) If true, this disk is thin provisioned, with
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

The `eagerly_scrub` and `thin_provisioned` control the space allocation type of
a virtual disk. These show up in the vSphere console as a unified enumeration
of options, the equivalents of which are explained below. The defaults in the
sub-resource are the equivalent of thin provisioning.

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
resource "vsphere_virtual_machine" "vm" {a
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
* `use_static_mac` - (Optional) If true, the `mac_address` field is treated as a
  static MAC address and set accordingly. Default: `false`.
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
resource "vsphere_virtual_machine" "vm" {a
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
  `CONTROLLER_TYPE:BUS_NUBER:UNIT_NUMBER`. Example: `scsi:0:1` means device
  unit 1 on SCSI bus 0.

### Creating a virtual machine from a template

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

#### Virtual machine customization

As part of the `clone` operation, a virtual machine can be
[customized][vmware-docs-customize] to configure host, network, or licensing
settings.

[vmware-docs-customize]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.vm_admin.doc/GUID-58E346FF-83AE-42B8-BE58-253641D257BC.html

To perform virtual machine customization as a part of the clone process,
specify the `customize` sub-resource within the `clone` sub-resource with the
respective customization options.  See the [cloning and customization
example](#cloning-and-customization-example) for a usage synopsis.

The settings for `customize` are as follows:



#### Additional requirements and notes for cloning

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
  disk controller type, so when using such OSes, make sure to ensure that
  `scsi_type` is set to an exact match of the template's controller set. For
  maximum compatibility, make sure the SCSI controllers on the source template
  are all the same type.

To ease the gathering of some of these options, you can use the
[`vsphere_virtual_machine` data source][tf-vsphere-virtual-machine-ds], which
will give you disk sizes, network interface types, SCSI bus types, and also the
guest ID of the source template.  See the [cloning and customization
example](#cloning-and-customization-example) for usage details.

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
