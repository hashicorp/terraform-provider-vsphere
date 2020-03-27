---
 layout: "vsphere"
 page_title: "VMware vSphere: vsphere_compute_policy"
 sidebar_current: "docs-vsphere-data-source-compute-policy"
 description: |-
   Provides a vSphere compute policy data source. This can be used to get the general attributes of a vSphere compute policy.
 ---

 # vsphere\_compute\_policy

 The `vsphere_compute_policy` data source can be used to discover the ID of a
 compute policy in vSphere. This is useful to fetch the ID of a policy that you want
 to use for virtual machine placement via the
 [`vsphere_virtual_machine`][docs-virtual-machine-resource] resource, allowing
 you to specify the cluster's root resource pool directly versus using the alias
 available through the [`vsphere_resource_pool`][docs-resource-pool-data-source]
 data source.

 [docs-virtual-machine-resource]: /docs/providers/vsphere/r/virtual_machine.html
 [docs-resource-pool-data-source]: /docs/providers/vsphere/d/resource_pool.html

 -> You may also wish to see the
 [`vsphere_compute_policy`][docs-compute-policy-resource] resource for further
 details about clusters or how to work with them in Terraform.

 [docs-compute-policy-resource]: /docs/providers/vsphere/r/compute_policy.html

 ## Example Usage

 ```hcl
 data "vsphere_compute_policy" "compute_policy" {
   name          = "compute-policy1"
 }
 ```

 ## Argument Reference

 The following arguments are supported:

 * `name` - (Required) The name compute policy.

 [docs-about-morefs]: /docs/providers/vsphere/index.html#use-of-managed-object-references-by-the-vsphere-provider

 ## Attribute Reference

 The following attributes are exported:

 * `id`: The [managed object reference ID][docs-about-morefs] of the policy.
 * `policy_type`: The type of of the compute policy. Will be one of
   `vm_host_affinity`, `vm_host_anti_affinity`, `vm_vm_affinity`, `vm_vm_anti_affinity`.
