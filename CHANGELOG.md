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

* resource/vsphere_virtual_machine: Fixed a regression with newtork discovery
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
