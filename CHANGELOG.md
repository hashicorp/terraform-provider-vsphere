## 0.3.1 (Unreleased)

FEATURES:

* **New Data Source:** `vsphere_tag` [GH-171]
* **New Data Source:** `vsphere_tag_category` [GH-167]
* **New Resoruce:** `vsphere_tag` [GH-171]
* **New Resoruce:** `vsphere_tag_category` [GH-164]

IMPROVEMENTS:

* resource/vsphere_virtual_machine: Adjusted the customization timeout to 10
  minutes [GH-168]

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
