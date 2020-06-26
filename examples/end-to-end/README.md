# vSphere 1.3.0 end-to-end Example

This repository is designed to demonstrate the capabilities of the [Terraform
vSphere Provider][ref-tf-vsphere] at the time of the 1.3.0 release.

[ref-tf-vsphere]: https://www.terraform.io/docs/providers/vsphere/index.html

This example performs the following:

* Sets up an NFS datastore across a number of hosts. This uses the
  [`vsphere_nas_datastore` resource][ref-tf-vsphere-nas-datastore].
* Sets up a vSphere distributed virtual switch (DVS) across a number of hosts,
  using the [`vsphere_distributed_virtual_switch` resource][ref-tf-vsphere-dvs].
* Creates a port group on the created DVS with a configured VLAN, using the
  [`vsphere_distributed_port_group` resource][ref-tf-vsphere-dvportgroup].
* Finally, creates a virtual machine using the [`vsphere_virtual_machine`
  resource][ref-tf-vsphere-virtual-machine] on the above three created
  resources.

[ref-tf-vsphere-nas-datastore]: https://www.terraform.io/docs/providers/vsphere/r/nas_datastore.html
[ref-tf-vsphere-dvs]: https://www.terraform.io/docs/providers/vsphere/r/distributed_virtual_switch.html
[ref-tf-vsphere-dvportgroup]: https://www.terraform.io/docs/providers/vsphere/r/distributed_port_group.html
[ref-tf-vsphere-virtual-machine]: https://www.terraform.io/docs/providers/vsphere/r/virtual_machine.html

Several data sources are also used:

* [`vsphere_datacenter`][ref-tf-vsphere-datacenter] - To get a datacenter
* [`vsphere_resource_pool`][ref-tf-vsphere-resource-pool] - To get a resource
  pool
* [`vsphere_virtual_machine`][ref-tf-vsphere-vm-data-source] - To get a virtual
  machine template.

[ref-tf-vsphere-datacenter]: https://www.terraform.io/docs/providers/vsphere/d/datacenter.html
[ref-tf-vsphere-resource-pool]: https://www.terraform.io/docs/providers/vsphere/d/resource_pool.html
[ref-tf-vsphere-vm-data-source]: https://www.terraform.io/docs/providers/vsphere/d/virtual_machine.html

## Requirements

* A working vCenter installation. This example does not support ESXi. You must
  have at least 3 hosts in a single cluster.
* Your ESXi hosts should have at least one free NIC available.
* The ESXi hosts should have access to an NFS server with an available share.
* A suitably modern Linux VM with one virtual disk. This needs to be
  customizable by your version of vSphere.

## Usage Details

You can either clone the entire
[terraform-provider-vsphere][ref-tf-vsphere-github] repository, or download the
`provider.tf`, `variables.tf`, `data_sources.tf`, `resources.tf`, and
`terraform.tfvars.example` files into a directory of your choice. Once done,
edit the `terraform.tfvars.example` file, populating the fields with the
relevant values, and then rename it to `terraform.tfvars`. Don't forget to
configure your endpoint and credentials by either adding them to the
`provider.tf` file, or by using enviornment variables. See
[here][ref-tf-vsphere-provider-settings] for a reference on provider-level
configuration values.

[ref-tf-vsphere-github]: https://github.com/hashicorp/terraform-provider-vsphere
[ref-tf-vsphere-provider-settings]: https://www.terraform.io/docs/providers/vsphere/index.html#argument-reference

Once done, run `terraform init`, and `terraform plan` to review the plan, then
`terraform apply` to execute. If you use Terraform 0.11.0 or higher, you can
skip `terraform plan` as `terraform apply` will now perform the plan for you and
ask you confirm the changes.

## Further Reading

This configuration is the working example for [this blog
post][a-re-introduction-to-the-terraform-vsphere-provider] on the [HashiCorp
website][hc-website].

[a-re-introduction-to-the-terraform-vsphere-provider]: https://www.hashicorp.com/blog/a-re-introduction-to-the-terraform-vsphere-provider
[hc-website]: https://www.hashicorp.com/
