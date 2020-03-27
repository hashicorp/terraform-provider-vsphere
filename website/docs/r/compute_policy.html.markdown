---
 subcategory: "Host and Cluster Management"
 layout: "vsphere"
 page_title: "VMware vSphere: vsphere_compute_policy"
 sidebar_current: "docs-vsphere-resource-compute-policy"
 description: |-
   Provides a VMware vSphere compute policy resource.
 ---

 # vsphere\_compute\_policy

 Provides a VMware vSphere compute policy resource.

 ~> **NOTE:** Compute policy is only supported for VMware Cloud on AWS
 customers as a tech preview feature.

 ## Example Usages

 **Create a vm_host_affinity compute policy:**

 ```hcl
 data "vsphere_tag_category" "category" {
 	name        = "test-category"
 }

 data "vsphere_tag" "tag" {
 	name        = "test-tag"
 	category_id = "${vsphere_tag_category.category.id}"
 }

 resource "vsphere_compute_policy" "policy_vm_host_affinity" {
 	name = "my_policy1"
 	description = "vm_host_affinity"
 	vm_tag = "${vsphere_tag.tag.id}"
 	host_tag = "${vsphere_tag.tag.id}"
 	capability = "vm_host_affinity"
 }
 ```

 ## Argument Reference

 The following arguments are supported:

 * `name` - (Required) The name of the compute policy. This name needs to be unique
   within the vCenter.
 * `description` - (Optional) The description of the compute policy.
 * `vm_tag` - (Required) The IDs of the tag that is associated to the VMs.
 * `host_tag` - (Optional) The IDs of the tag that is associated to the Hosts. It's required for
 vm_host_affinity or vm_host_anti_affinity policies and optional for vm_vm_affinity or vm_vm_anti_affinity
 policies.

 ~> **NOTE:** Tagging support is unsupported on direct ESXi connections and
 requires vCenter 6.0 or higher.

 * `capability` - (Required) The type of the compute policy. Will be one of
   `vm_host_affinity`, `vm_host_anti_affinity`, `vm_vm_affinity`, `vm_vm_anti_affinity`.

 ## Attribute Reference

 * `moid` - [Managed object ID][docs-about-morefs] of this compute policy.

 [docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider
