## 0.2.0 (Unreleased)

BREAKING CHANGES:

* resource/vsphere_virtual_disk: Default adapter type is now `lsiLogic`,
  changed from `ide`. [GH-94]

FEATURES:

* **New Resource:** `vsphere_datacenter` [GH-126]
* **New Resource:** `vsphere_license` [GH-110]

IMPROVEMENTS:

* resource/vsphere_virtual_machine: Add annotation argument [GH-111]

BUG FIXES:

* Updated [govmomi](https://github.com/vmware/govmomi) to v0.15.0 [GH-114]
* Updated network interface discovery behaviour in refresh. [GH-129]. This fixes
  several reported bugs - see the PR for references!

## 0.1.0 (June 20, 2017)

NOTES:

* Same functionality as that of Terraform 0.9.8. Repacked as part of [Provider Splitout](https://www.hashicorp.com/blog/upcoming-provider-changes-in-terraform-0-10/)
