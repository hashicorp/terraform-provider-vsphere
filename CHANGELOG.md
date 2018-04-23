## 1.4.1 (April 23, 2018)

IMPROVEMENTS:

* `resource/vsphere_virtual_machine`: Introduced the
  `wait_for_guest_net_routable` setting, which controls whether or not the guest
  network waiter waits on an address that matches the virtual machine's
  configured default gateway. ([#470](https://github.com/terraform-providers/terraform-provider-vsphere/issues/470))

BUG FIXES:

* `resource/vsphere_virtual_machine`: The resource now correctly blocks `clone`
  workflows on direct ESXi connections, where cloning is not supported. ([#476](https://github.com/terraform-providers/terraform-provider-vsphere/issues/476))
* `resource/vsphere_virtual_machine`: Corrected an issue that was preventing VMs
  from being migrated from one cluster to another. ([#474](https://github.com/terraform-providers/terraform-provider-vsphere/issues/474))
* `resource/vsphere_virtual_machine`: Corrected an issue where changing
  datastore information and cloning/customization parameters (which forces a new
  resource) at the same time was creating a diff mismatch after destroying the
  old virtual machine. ([#469](https://github.com/terraform-providers/terraform-provider-vsphere/issues/469))
* `resource/vsphere_virtual_machine`: Corrected a crash that can come up from an
  incomplete lookup of network information during network device management.
  ([#456](https://github.com/terraform-providers/terraform-provider-vsphere/issues/456))
* `resource/vsphere_virtual_machine`: Corrected some issues where some
  post-clone configuration errors were leaving the resource half-completed and
  irrecoverable without direct modification of the state. ([#467](https://github.com/terraform-providers/terraform-provider-vsphere/issues/467))
* `resource/vsphere_virtual_machine`: Corrected a crash that can come up when a
  retrieved virtual machine has no lower-level configuration object in the API.
  ([#463](https://github.com/terraform-providers/terraform-provider-vsphere/issues/463))

## 1.4.0 (April 10, 2018)

FEATURES:

* **New Resource:** `vsphere_storage_drs_vm_override` ([#450](https://github.com/terraform-providers/terraform-provider-vsphere/issues/450))
* **New Resource:** `vsphere_datastore_cluster` ([#436](https://github.com/terraform-providers/terraform-provider-vsphere/issues/436))
* **New Data Source:** `vsphere_datastore_cluster` ([#437](https://github.com/terraform-providers/terraform-provider-vsphere/issues/437))

IMPROVEMENTS:

* The provider now has the ability to persist sessions to disk, which can help
  when running large amounts of consecutive or concurrent Terraform operations
  at once. See the [provider
  documentation](https://www.terraform.io/docs/providers/vsphere/index.html) for
  more details. ([#422](https://github.com/terraform-providers/terraform-provider-vsphere/issues/422))
* `resource/vsphere_virtual_machine`: This resource now supports import of
  resources or migrations from legacy versions of the provider (provider version
  0.4.2 or earlier) into configurations that have the `clone` block specified.
  See [Additional requirements and notes for
  importing](https://www.terraform.io/docs/providers/vsphere/r/virtual_machine.html#additional-requirements-and-notes-for-importing)
  in the resource documentation for more details. ([#460](https://github.com/terraform-providers/terraform-provider-vsphere/issues/460))
* `resource/vsphere_virtual_machine`: Now supports datastore clusters. Virtual
  machines placed in a datastore cluster will use Storage DRS recommendations
  for initial placement, virtual disk creation, and migration between datastore
  clusters. Migrations made by Storage DRS outside of Terraform will no longer
  create diffs when datastore clusters are in use. ([#447](https://github.com/terraform-providers/terraform-provider-vsphere/issues/447))
* `resource/vsphere_virtual_machine`: Added support for ISO transport of vApp
  properties. The resource should now behave better with virtual machines cloned
  from OVF/OVA templates that use the ISO transport to supply configuration
  settings. ([#381](https://github.com/terraform-providers/terraform-provider-vsphere/issues/381))
* `resource/vsphere_virtual_machine`: Added support for client mapped CDROM
  devices. ([#421](https://github.com/terraform-providers/terraform-provider-vsphere/issues/421))
* `resource/vsphere_virtual_machine`: Destroying a VM that currently has
  external disks attached should now function correctly and not give a duplicate
  UUID error. ([#442](https://github.com/terraform-providers/terraform-provider-vsphere/issues/442))
* `resource/vsphere_nas_datastore`: Now supports datastore clusters. ([#439](https://github.com/terraform-providers/terraform-provider-vsphere/issues/439))
* `resource/vsphere_vmfs_datastore`: Now supports datastore clusters. ([#439](https://github.com/terraform-providers/terraform-provider-vsphere/issues/439))

## 1.3.3 (March 01, 2018)

IMPROVEMENTS:

* `resource/vsphere_virtual_machine`: The `moid` attribute has now be re-added
  to the resource, exporting the managed object ID of the virtual machine.
  ([#390](https://github.com/terraform-providers/terraform-provider-vsphere/issues/390))

BUG FIXES:

* `resource/vsphere_virtual_machine`: Fixed a crash scenario that can happen
  when a virtual machine is deployed to a cluster that does not have any hosts,
  or under certain circumstances such an expired vCenter license. ([#414](https://github.com/terraform-providers/terraform-provider-vsphere/issues/414))
* `resource/vsphere_virtual_machine`: Corrected an issue reading disk capacity
  values after a vCenter or ESXi upgrade. ([#405](https://github.com/terraform-providers/terraform-provider-vsphere/issues/405))
* `resource/vsphere_virtual_machine`: Opaque networks, such as those coming from
  NSX, should now be able to be correctly added as networks for virtual
  machines. ([#398](https://github.com/terraform-providers/terraform-provider-vsphere/issues/398))

## 1.3.2 (February 07, 2018)

BUG FIXES:

* `resource/vsphere_virtual_machine`: Changed the update implemented in ([#377](https://github.com/terraform-providers/terraform-provider-vsphere/issues/377))
  to use a local filter implementation. This corrects situations where virtual
  machines in inventory with orphaned or otherwise corrupt configurations were
  interfering with UUID searches, creating erroneous duplicate UUID errors. This
  fix applies to vSphere 6.0 and lower only. vSphere 6.5 was not affected.
  ([#391](https://github.com/terraform-providers/terraform-provider-vsphere/issues/391))

## 1.3.1 (February 01, 2018)

BUG FIXES:

* `resource/vsphere_virtual_machine`: Looking up templates by their UUID now
  functions correctly for vSphere 6.0 and earlier. ([#377](https://github.com/terraform-providers/terraform-provider-vsphere/issues/377))

## 1.3.0 (January 26, 2018)

BREAKING CHANGES:

* The `vsphere_virtual_machine` resource now has a new method of identifying
  virtual disk sub-resources, via the `label` attribute. This replaces the
  `name` attribute, which has now been marked as deprecated and will be removed
  in the next major version (2.0.0). Further to this, there is a `path`
  attribute that now must also be supplied for external disks. This has lifted
  several virtual disk-related cloning and migration restrictions, in addition
  to changing requirements for importing. See the [resource
  documentation](https://www.terraform.io/docs/providers/vsphere/r/virtual_machine.html)
  for usage details.

IMPROVEMENTS:

* `resource/vsphere_virtual_machine`: Fixed an issue where certain changes
  happening at the same time (such as a disk resize along with a change of SCSI
  type) were resulting in invalid device change operations. ([#371](https://github.com/terraform-providers/terraform-provider-vsphere/issues/371))
* `resource/vsphere_virtual_machine`: Introduced the `label` argument, which
  allows one to address a virtual disk independent of its VMDK file name and
  position on the SCSI bus. ([#363](https://github.com/terraform-providers/terraform-provider-vsphere/issues/363))
* `resource/vsphere_virtual_machine`: Introduced the `path` argument, which
  replaces the `name` attribute for supplying the path for externally attached
  disks supplied with `attach = true`, and is otherwise a computed attribute
  pointing to the current path of any specific virtual disk. ([#363](https://github.com/terraform-providers/terraform-provider-vsphere/issues/363))
* `resource/vsphere_virtual_machine`: Introduced the `uuid` attribute, a new
  computed attribute that allows the tracking of a disk independent of its
  current position on the SCSI bus. This is used in all scenarios aside from
  freshly-created or added virtual disks. ([#363](https://github.com/terraform-providers/terraform-provider-vsphere/issues/363))
* `resource/vsphere_virtual_machine`: The virtual disk `name` argument is now
  deprecated and will be removed from future releases. It no longer dictates the
  name of non-attached VMDK files and serves as an alias to the now-split `label`
  and `path` attributes. ([#363](https://github.com/terraform-providers/terraform-provider-vsphere/issues/363))
* `resource/vsphere_virtual_machine`: Cloning no longer requires you to choose a
  disk label (name) that matches the name of the VM. ([#363](https://github.com/terraform-providers/terraform-provider-vsphere/issues/363))
* `resource/vsphere_virtual_machine`: Storage vMotion can now be performed on
  renamed virtual machines. ([#363](https://github.com/terraform-providers/terraform-provider-vsphere/issues/363))
* `resource/vsphere_virtual_machine`: Storage vMotion no longer cares what your
  disks are labeled (named), and will not block migrations based on the naming
  criteria added after 1.1.1. ([#363](https://github.com/terraform-providers/terraform-provider-vsphere/issues/363))
* `resource/vsphere_virtual_machine`: Storage vMotion now works on linked
  clones. ([#363](https://github.com/terraform-providers/terraform-provider-vsphere/issues/363))
* `resource/vsphere_virtual_machine`: The import restrictions for virtual disks
  have changed, and rather than ensuring that disk `name` arguments match a
  certain convention, `label` is now expected to match a convention of `diskN`,
  where N is the disk number, ordered by the disk's position on the SCSI bus.
  Importing to a configuration still using `name` to address disks is no longer
  supported. ([#363](https://github.com/terraform-providers/terraform-provider-vsphere/issues/363))
* `resource/vsphere_virtual_machine`: Now supports setting vApp properties that
  usually come from an OVF/OVA template or virtual appliance. ([#303](https://github.com/terraform-providers/terraform-provider-vsphere/issues/303))

## 1.2.0 (January 11, 2018)

FEATURES:

* **New Resource:** `vsphere_custom_attribute` ([#229](https://github.com/terraform-providers/terraform-provider-vsphere/issues/229))
* **New Data Source:** `vsphere_custom_attribute` ([#229](https://github.com/terraform-providers/terraform-provider-vsphere/issues/229))

IMPROVEMENTS:

* All vSphere provider resources that are capable of doing so now support custom
  attributes. Check the documentation of any specific resource for more details!
  ([#229](https://github.com/terraform-providers/terraform-provider-vsphere/issues/229))
* `resource/vsphere_virtual_machine`: The resource will now disallow a disk's
  `name` coming from a value that is still unavailable at plan time (such as a
  computed value from a resource). ([#329](https://github.com/terraform-providers/terraform-provider-vsphere/issues/329))

BUG FIXES:

* `resource/vsphere_virtual_machine`: Fixed an issue that was causing crashes
  when working with virtual machines or templates when no network interface was
  occupying the first available device slot on the PCI bus. ([#344](https://github.com/terraform-providers/terraform-provider-vsphere/issues/344))

## 1.1.1 (December 14, 2017)

IMPROVEMENTS:

* `resource/vsphere_virtual_machine`: Network interface resource allocation
  options are now restricted to vSphere 6.0 and higher, as they are unsupported
  on vSphere 5.5. ([#322](https://github.com/terraform-providers/terraform-provider-vsphere/issues/322))
* `resource/vsphere_virtual_machine`: Resources that were deleted outside of
  Terraform will now be marked as gone in the state, causing them to be
  re-created during the next apply. ([#321](https://github.com/terraform-providers/terraform-provider-vsphere/issues/321))
* `resource/vsphere_virtual_machine`: Added some restrictions to storage vMotion
  to cover some currently un-supported scenarios that were still allowed,
  leading to potentially dangerous situations or invalid post-application
  states. ([#319](https://github.com/terraform-providers/terraform-provider-vsphere/issues/319))
* `resource/vsphere_virtual_machine`: The resource now treats disks that it does
  not recognize at a known device address as orphaned, and will set
  `keep_on_remove` to safely remove them. ([#317](https://github.com/terraform-providers/terraform-provider-vsphere/issues/317))
* `resource/vsphere_virtual_machine`: The resource now attempts to detect unsafe
  disk deletion scenarios that can happen from the renaming of a virtual machine
  in situations where the VM and disk names may share a common variable. The
  provider will block such operations from proceeding. ([#305](https://github.com/terraform-providers/terraform-provider-vsphere/issues/305))

## 1.1.0 (December 07, 2017)

BREAKING CHANGES:

* The `vsphere_virtual_machine` _data source_ has a new sub-resource attribute
  for disk information, named `disks`. This takes the place of `disk_sizes`,
  which has been moved to a `size` attribute within this new sub-resource, and
  also contains information about the discovered disks' `eagerly_scrub` and
  `thin_provisioned` settings. This is to facilitate the ability to discover all
  settings that could cause issues when cloning virtual machines.

To transition to the new syntax, any `disk` sub-resource in a
`vsphere_virtual_machine` resource that depends on a syntax such as:

```
resource "vsphere_virtual_machine" "vm" {
  ...

  disk {
    name = "terraform-test.vmdk"
    size = "${data.vsphere_virtual_machine.template.disk_sizes[0]}"
  }
}
```

Should be changed to:

```
resource "vsphere_virtual_machine" "vm" {
  ...

  disk {
    name = "terraform-test.vmdk"
    size = "${data.vsphere_virtual_machine.template.disks.0.size}"
  }
}
```

If you are using `linked_clone`, add the new settings for `eagerly_scrub` and
`thin_provisioned`:

```
resource "vsphere_virtual_machine" "vm" {
  ...

  disk {
    name             = "terraform-test.vmdk"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }
}
```

For a more complete example, see the [cloning and customization
example](https://www.terraform.io/docs/providers/vsphere/r/virtual_machine.html#cloning-and-customization-example)
in the documentation.

BUG FIXES:

* `resource/vsphere_virtual_machine`: Fixed a bug with NIC device assignment
  logic that was causing a crash when adding more than 3 NICs to a VM. ([#280](https://github.com/terraform-providers/terraform-provider-vsphere/issues/280))
* `resource/vsphere_virtual_machine`: CDROM devices on cloned virtual machines
  are now connected properly on power on. ([#278](https://github.com/terraform-providers/terraform-provider-vsphere/issues/278))
* `resource/vsphere_virtual_machine`: Tightened the pre-clone checks for virtual
  disks to ensure that the size and disk types are the same between the template
  and the created virtual machine's configuration. ([#277](https://github.com/terraform-providers/terraform-provider-vsphere/issues/277))

## 1.0.3 (December 06, 2017)

BUG FIXES:

* `resource/vsphere_virtual_machine`: Fixed an issue in the post-clone process
  when a CDROM device exists in configuration. ([#276](https://github.com/terraform-providers/terraform-provider-vsphere/issues/276))

## 1.0.2 (December 05, 2017)

BUG FIXES:

* `resource/vsphere_virtual_machine`: Fixed issues related to correct processing
  VM templates with no network interfaces, or fewer network interfaces than the
  amount that will ultimately end up in configuration. ([#269](https://github.com/terraform-providers/terraform-provider-vsphere/issues/269))
* `resource/vsphere_virtual_machine`: Version comparison logic now functions
  correctly to properly disable certain features when using older versions of
  vSphere. ([#272](https://github.com/terraform-providers/terraform-provider-vsphere/issues/272))

## 1.0.1 (December 02, 2017)

BUG FIXES:

* `resource/vsphere_virtual_machine`: Corrected an issue that was preventing the
  use of this resource on standalone ESXi. ([#263](https://github.com/terraform-providers/terraform-provider-vsphere/issues/263))
* `data/vsphere_resource_pool`: This data source now works as documented on
  standalone ESXi. ([#263](https://github.com/terraform-providers/terraform-provider-vsphere/issues/263))

## 1.0.0 (December 01, 2017)

BREAKING CHANGES:

* The `vsphere_virtual_machine` resource has received a major update and change
  to its interface. See the documentation for the resource for full details,
  including information on things to consider while migrating the new version of
  the resource.

FEATURES:

* **New Data Source:** `vsphere_resource_pool` ([#244](https://github.com/terraform-providers/terraform-provider-vsphere/issues/244))
* **New Data Source:** `vsphere_datastore` ([#244](https://github.com/terraform-providers/terraform-provider-vsphere/issues/244))
* **New Data Source:** `vsphere_virtual_machine` ([#244](https://github.com/terraform-providers/terraform-provider-vsphere/issues/244))

IMPROVEMENTS:

* `resource/vsphere_virtual_machine`: The distinct VM workflows are now better
  defined: all cloning options are now contained within a `clone` sub-resource,
  with customization being a `customize` sub-resource off of that. Absence of
  the `clone` sub-resource means no cloning or customization will occur.
  ([#244](https://github.com/terraform-providers/terraform-provider-vsphere/issues/244))
* `resource/vsphere_virtual_machine`: Nearly all customization options have now
  been exposed. Magic values such as hostname and DNS defaults have been
  removed, with some of these options now being required values depending on the
  OS being customized. ([#244](https://github.com/terraform-providers/terraform-provider-vsphere/issues/244))
* `resource/vsphere_virtual_machine`: Device management workflows have been
  greatly improved, exposing more options and fixing several bugs. ([#244](https://github.com/terraform-providers/terraform-provider-vsphere/issues/244))
* `resource/vsphere_virtual_machine`: Added support for CPU and memory hot-plug.
  Several other VM reconfiguration operations are also supported while the VM is
  powered on, guest type and VMware tools permitting in some cases. ([#244](https://github.com/terraform-providers/terraform-provider-vsphere/issues/244))
* `resource/vsphere_virtual_machine`: The resource now supports both host and
  storage vMotion. Virtual machines can now be moved between hosts, clusters,
  resource pools, and datastores. Individual disks can be pinned to a single
  datastore with a VM located in another. ([#244](https://github.com/terraform-providers/terraform-provider-vsphere/issues/244))
* `resource/vsphere_virtual_machine`: The resource now supports import. ([#244](https://github.com/terraform-providers/terraform-provider-vsphere/issues/244))
* `resource/vsphere_virtual_machine`: Several other minor improvements, see
  documentation for more details. ([#244](https://github.com/terraform-providers/terraform-provider-vsphere/issues/244))

BUG FIXES:

* `resource/vsphere_virtual_machine`: Several long-standing issues have been fixed,
  namely surrounding virtual disk and network device management. ([#244](https://github.com/terraform-providers/terraform-provider-vsphere/issues/244))
* `resource/vsphere_host_virtual_switch`: This resource now correctly supports a
  configuration with no NICs. ([#256](https://github.com/terraform-providers/terraform-provider-vsphere/issues/256))
* `data/vsphere_network`: No longer restricted to being used on vCenter. ([#248](https://github.com/terraform-providers/terraform-provider-vsphere/issues/248))

## 0.4.2 (October 13, 2017)

FEATURES:

* **New Data Source:** `vsphere_network` ([#201](https://github.com/terraform-providers/terraform-provider-vsphere/issues/201))
* **New Data Source:** `vsphere_distributed_virtual_switch` ([#170](https://github.com/terraform-providers/terraform-provider-vsphere/issues/170))
* **New Resource:** `vsphere_distributed_port_group` ([#189](https://github.com/terraform-providers/terraform-provider-vsphere/issues/189))
* **New Resource:** `vsphere_distributed_virtual_switch` ([#188](https://github.com/terraform-providers/terraform-provider-vsphere/issues/188))

IMPROVEMENTS:

* resource/vsphere_virtual_machine: The customization waiter is now tunable
  through the `wait_for_customization_timeout` argument. The timeout can be
  adjusted or the waiter can be disabled altogether. ([#199](https://github.com/terraform-providers/terraform-provider-vsphere/issues/199))
* resource/vsphere_virtual_machine: `domain` now acts as a default for
  `dns_suffixes` if the latter is not defined, setting the value in `domain` as
  a search domain in the customization specification. `vsphere.local` is not
  used as a last resort only. ([#185](https://github.com/terraform-providers/terraform-provider-vsphere/issues/185))
* resource/vsphere_virtual_machine: Expose the `adapter_type` parameter to allow
  the control of the network interface type. This is currently restricted to
  `vmxnet3` and `e1000` but offers more control than what was available before,
  and more interface types will follow in later versions of the provider.
  ([#193](https://github.com/terraform-providers/terraform-provider-vsphere/issues/193))

BUG FIXES:

* resource/vsphere_virtual_machine: Fixed a regression with network discovery
  that was causing Terraform to crash while the VM was in a powered off state.
  ([#198](https://github.com/terraform-providers/terraform-provider-vsphere/issues/198))
* All resources that can use tags will now properly remove their tags completely
  (or remove any out-of-band added tags) when the `tags` argument is not present
  in configuration. ([#196](https://github.com/terraform-providers/terraform-provider-vsphere/issues/196))

## 0.4.1 (October 02, 2017)

BUG FIXES:

* resource/vsphere_folder: Migration of state from a version of this resource
  before v0.4.0 now works correctly. ([#187](https://github.com/terraform-providers/terraform-provider-vsphere/issues/187))

## 0.4.0 (September 29, 2017)

BREAKING CHANGES:

* The `vsphere_folder` resource has been re-written, and its configuration is
  significantly different. See the [resource
  documentation](https://www.terraform.io/docs/providers/vsphere/r/folder.html)
  for more details. Existing state will be migrated. ([#179](https://github.com/terraform-providers/terraform-provider-vsphere/issues/179))

FEATURES:

* **New Data Source:** `vsphere_tag` ([#171](https://github.com/terraform-providers/terraform-provider-vsphere/issues/171))
* **New Data Source:** `vsphere_tag_category` ([#167](https://github.com/terraform-providers/terraform-provider-vsphere/issues/167))
* **New Resoruce:** `vsphere_tag` ([#171](https://github.com/terraform-providers/terraform-provider-vsphere/issues/171))
* **New Resoruce:** `vsphere_tag_category` ([#164](https://github.com/terraform-providers/terraform-provider-vsphere/issues/164))

IMPROVEMENTS:

* resource/vsphere_folder: You can now create any kind of folder with this
  resource, not just virtual machine folders. ([#179](https://github.com/terraform-providers/terraform-provider-vsphere/issues/179))
* resource/vsphere_folder: Now supports tags. ([#179](https://github.com/terraform-providers/terraform-provider-vsphere/issues/179))
* resource/vsphere_folder: Now supports import. ([#179](https://github.com/terraform-providers/terraform-provider-vsphere/issues/179))
* resource/vsphere_datacenter: Tags can now be applied to datacenters. ([#177](https://github.com/terraform-providers/terraform-provider-vsphere/issues/177))
* resource/vsphere_nas_datastore: Tags can now be applied to NAS datastores.
  ([#176](https://github.com/terraform-providers/terraform-provider-vsphere/issues/176))
* resource/vsphere_vmfs_datastore: Tags can now be applied to VMFS datastores.
  ([#176](https://github.com/terraform-providers/terraform-provider-vsphere/issues/176))
* resource/vsphere_virtual_machine: Tags can now be applied to virtual machines.
  ([#175](https://github.com/terraform-providers/terraform-provider-vsphere/issues/175))
* resource/vsphere_virtual_machine: Adjusted the customization timeout to 10
  minutes ([#168](https://github.com/terraform-providers/terraform-provider-vsphere/issues/168))

BUG FIXES:

* resource/vsphere_virtual_machine: This resource can now be used with networks
  with unescaped slashes in its network name. ([#181](https://github.com/terraform-providers/terraform-provider-vsphere/issues/181))
* resource/vsphere_virtual_machine: Fixed a crash where virtual NICs were
  created with networks backed by a 3rd party hardware VDS. ([#181](https://github.com/terraform-providers/terraform-provider-vsphere/issues/181))
* resource/vsphere_virtual_machine: Fixed crashes and spurious diffs that were
  caused by errors in the code that associates the default gateway with its
  correct network device during refresh. ([#180](https://github.com/terraform-providers/terraform-provider-vsphere/issues/180))

## 0.3.0 (September 14, 2017)

BREAKING CHANGES:

* `vsphere_virtual_machine` now waits on a _routeable_ IP address by default,
  and does not wait when running `terraform plan`, `terraform refresh`, or
  `terraform destroy`. There is also now a timeout of 5 minutes, after which
  `terraform apply` will fail with an error. Note that the apply may not fail
  exactly on the 5 minute mark. The network waiter can be disabled completely by
  setting `wait_for_guest_net` to `false`. ([#158](https://github.com/terraform-providers/terraform-provider-vsphere/issues/158))

FEATURES:

* **New Resource:** `vsphere_virtual_machine_snapshot` ([#107](https://github.com/terraform-providers/terraform-provider-vsphere/issues/107))

IMPROVEMENTS:

* resource/vsphere_virtual_machine: Virtual machine power state is now enforced.
  Terraform will trigger a diff if the VM is powered off or suspended, and power
  it back on during the next apply. ([#152](https://github.com/terraform-providers/terraform-provider-vsphere/issues/152))

BUG FIXES:

* resource/vsphere_virtual_machine: Fixed customization behavior to watch
  customization events for success, rather than returning immediately when the
  `CustomizeVM` task returns. This is especially important during Windows
  customization where a large part of the customization task involves
  out-of-band configuration through Sysprep. ([#158](https://github.com/terraform-providers/terraform-provider-vsphere/issues/158))

## 0.2.2 (September 07, 2017)

FEATURES:

* **New Resource:** `vsphere_nas_datastore` ([#149](https://github.com/terraform-providers/terraform-provider-vsphere/issues/149))
* **New Resource:** `vsphere_vmfs_datastore` ([#142](https://github.com/terraform-providers/terraform-provider-vsphere/issues/142))
* **New Data Source:** `vsphere_vmfs_disks` ([#141](https://github.com/terraform-providers/terraform-provider-vsphere/issues/141))

## 0.2.1 (August 31, 2017)

FEATURES:

* **New Resource:** `vsphere_host_port_group` ([#139](https://github.com/terraform-providers/terraform-provider-vsphere/issues/139))
* **New Resource:** `vsphere_host_virtual_switch` ([#138](https://github.com/terraform-providers/terraform-provider-vsphere/issues/138))
* **New Data Source:** `vsphere_datacenter` ([#144](https://github.com/terraform-providers/terraform-provider-vsphere/issues/144))
* **New Data Source:** `vsphere_host` ([#146](https://github.com/terraform-providers/terraform-provider-vsphere/issues/146))

IMPROVEMENTS:

* resource/vsphere_virtual_machine: Allow customization of hostname ([#79](https://github.com/terraform-providers/terraform-provider-vsphere/issues/79))

BUG FIXES:

* resource/vsphere_virtual_machine: Fix IPv4 address mapping issues causing
  spurious diffs, in addition to IPv6 normalization issues that can lead to spurious
  diffs as well. ([#128](https://github.com/terraform-providers/terraform-provider-vsphere/issues/128))

## 0.2.0 (August 23, 2017)

BREAKING CHANGES:

* resource/vsphere_virtual_disk: Default adapter type is now `lsiLogic`,
  changed from `ide`. ([#94](https://github.com/terraform-providers/terraform-provider-vsphere/issues/94))

FEATURES:

* **New Resource:** `vsphere_datacenter` ([#126](https://github.com/terraform-providers/terraform-provider-vsphere/issues/126))
* **New Resource:** `vsphere_license` ([#110](https://github.com/terraform-providers/terraform-provider-vsphere/issues/110))

IMPROVEMENTS:

* resource/vsphere_virtual_machine: Add annotation argument ([#111](https://github.com/terraform-providers/terraform-provider-vsphere/issues/111))

BUG FIXES:

* Updated [govmomi](https://github.com/vmware/govmomi) to v0.15.0 ([#114](https://github.com/terraform-providers/terraform-provider-vsphere/issues/114))
* Updated network interface discovery behaviour in refresh. [[#129](https://github.com/terraform-providers/terraform-provider-vsphere/issues/129)]. This fixes
  several reported bugs - see the PR for references!

## 0.1.0 (June 20, 2017)

NOTES:

* Same functionality as that of Terraform 0.9.8. Repacked as part of [Provider Splitout](https://www.hashicorp.com/blog/upcoming-provider-changes-in-terraform-0-10/)
