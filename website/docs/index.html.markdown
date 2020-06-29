---
layout: "vsphere"
page_title: "Provider: VMware vSphere"
sidebar_current: "docs-vsphere-index"
description: |-
  A Terraform provider to work with VMware vSphere, allowing management of virtual machines and other VMware resources. Supports management through both vCenter and ESXi.
---

# VMware vSphere Provider

The VMware vSphere provider gives Terraform the ability to work with VMware vSphere
Products, notably [vCenter Server][vmware-vcenter] and [ESXi][vmware-esxi].
This provider can be used to manage many aspects of a VMware vSphere
environment, including virtual machines, standard and distributed networks,
datastores, and more.

[vmware-vcenter]: https://www.vmware.com/products/vcenter-server.html
[vmware-esxi]: https://www.vmware.com/products/esxi-and-esx.html

Use the navigation on the left to read about the various resources and data
sources supported by the provider.

~> **NOTE:** This provider requires API write access and hence is not supported
on a free ESXi license.

## Example Usage

The following abridged example demonstrates a current basic usage of the
provider to launch a virtual machine using the [`vsphere_virtual_machine`
resource][tf-vsphere-virtual-machine-resource]. The datacenter, datastore,
resource pool, and network are discovered via the
[`vsphere_datacenter`][tf-vsphere-datacenter],
[`vsphere_datastore`][tf-vsphere-datastore],
[`vsphere_resource_pool`][tf-vsphere-resource-pool], and
[`vsphere_network`][tf-vsphere-network] data sources respectively. Most of
these resources can be directly managed by Terraform as well - check the
sidebar for specific resources.

[tf-vsphere-virtual-machine-resource]: /docs/providers/vsphere/r/virtual_machine.html
[tf-vsphere-datacenter]: /docs/providers/vsphere/d/datacenter.html
[tf-vsphere-datastore]: /docs/providers/vsphere/d/datastore.html
[tf-vsphere-resource-pool]: /docs/providers/vsphere/d/resource_pool.html
[tf-vsphere-network]: /docs/providers/vsphere/d/network.html

```hcl
provider "vsphere" {
  user           = var.vsphere_user
  password       = var.vsphere_password
  vsphere_server = var.vsphere_server

  # If you have a self-signed cert
  allow_unverified_ssl = true
}

data "vsphere_datacenter" "dc" {
  name = "dc1"
}

data "vsphere_datastore" "datastore" {
  name          = "datastore1"
  datacenter_id = data.vsphere_datacenter.dc.id
}

data "vsphere_resource_pool" "pool" {
  name          = "cluster1/Resources"
  datacenter_id = data.vsphere_datacenter.dc.id
}

data "vsphere_network" "network" {
  name          = "public"
  datacenter_id = data.vsphere_datacenter.dc.id
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = data.vsphere_resource_pool.pool.id
  datastore_id     = data.vsphere_datastore.datastore.id

  num_cpus = 2
  memory   = 1024
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = data.vsphere_network.network.id
  }

  disk {
    label = "disk0"
    size  = 20
  }
}
```

See the sidebar for usage information on all of the resources, which will have
examples specific to their own use cases.

## Argument Reference

The following arguments are used to configure the VMware vSphere Provider:

* `user` - (Required) This is the username for vSphere API operations. Can also
  be specified with the `VSPHERE_USER` environment variable.
* `password` - (Required) This is the password for vSphere API operations. Can
  also be specified with the `VSPHERE_PASSWORD` environment variable.
* `vsphere_server` - (Required) This is the vCenter server name for vSphere API
  operations. Can also be specified with the `VSPHERE_SERVER` environment
  variable.
* `allow_unverified_ssl` - (Optional) Boolean that can be set to true to
  disable SSL certificate verification. This should be used with care as it
  could allow an attacker to intercept your auth token. If omitted, default
  value is `false`. Can also be specified with the `VSPHERE_ALLOW_UNVERIFIED_SSL`
  environment variable.
* `vim_keep_alive` - (Optional) Keep alive interval in minutes for the VIM
  session. Standard session timeout in vSphere is 30 minutes. This defaults to
  10 minutes to ensure that operations that take a longer than 30 minutes
  without API interaction do not result in a session timeout. Can also be
  specified with the `VSPHERE_VIM_KEEP_ALIVE` environment variable.

### Session persistence options

The provider also provides session persistence options that can be configured
below. These can help when using Terraform in a way where session limits could
be normally reached by creating a new session for every run, such as a large
amount of concurrent or consecutive Terraform runs in a short period of time.

~> **NOTE:** Session keys are as good as user credentials for as long as the
session is valid for - handle them with care and delete them when you know you
will no longer need them.

* `persist_session` - (Optional) Persist the SOAP and REST client sessions to
  disk. Default: `false`. Can also be specified by the
  `VSPHERE_PERSIST_SESSION` environment variable.
* `vim_session_path` - (Optional) The direcotry to save the VIM SOAP API
  session to. Default: `${HOME}/.govmomi/sessions`. Can also be specified by
  the `VSPHERE_VIM_SESSION_PATH` environment variable.
* `rest_session_path` - The directory to save the REST API session to. 
  Default: `${HOME}/.govmomi/rest_sessions`. Can also be specified by the 
  `VSPHERE_REST_SESSION_PATH` environment variable.

#### govc/Terraform session interoperability

Note that the session format used to save VIM SOAP sessions is the same used
with [govc][docs-govc]. If you use govc as part of your provisioning
process, Terraform will use the saved session if present and if
`persist_session` is enabled.

### Debugging options

~> **NOTE:** The following options can leak sensitive data and should only be
enabled when instructed to do so by HashiCorp for the purposes of
troubleshooting issues with the provider, or when attempting to perform your
own troubleshooting. Use them at your own risk and do not leave them enabled!

* `client_debug` - (Optional) When `true`, the provider logs SOAP calls made to
  the vSphere API to disk.  The log files are logged to `${HOME}/.govmomi`.
  Can also be specified with the `VSPHERE_CLIENT_DEBUG` environment variable.
* `client_debug_path` - (Optional) Override the default log path. Can also
   be specified with the `VSPHERE_CLIENT_DEBUG_PATH` environment variable.
* `client_debug_path_run` - (Optional) A specific subdirectory in
  `client_debug_path` to use for debugging calls for this particular Terraform
  configuration. All data in this directory is removed at the start of the
  Terraform run. Can also be specified with the `VSPHERE_CLIENT_DEBUG_PATH_RUN`
  environment variable.

## Notes on Required Privileges

When using a non-administrator account to perform Terraform tasks, keep in mind
that most Terraform resources perform operations in a CRUD-like fashion and
require both read and write privileges to the resources they are managing. Make
sure that the user has appropriate read-write access to the resources you need
to work with. Read-only access should be sufficient when only using data
sources on some features. You can read more about vSphere permissions and user
management [here][vsphere-docs-user-management].

[vsphere-docs-user-management]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.security.doc/GUID-5372F580-5C23-4E9C-8A4E-EF1B4DD9033E.html

There are a couple of exceptions to keep in mind when setting up a restricted
provisioning user:

### Tags

If your vSphere version supports [tags][vsphere-docs-tags], keep in mind that
Terraform will always attempt to read tags from a resource, even if you do not
have any tags defined. Ensure that your user has access to at least read tags,
or else you will encounter errors.

[vsphere-docs-tags]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.vcenterhost.doc/GUID-E8E854DD-AA97-4E0C-8419-CE84F93C4058.html

### Events

Likewise, some Terraform resources will attempt to read event data from vSphere
to check for certain events (such as virtual machine customization or power
events). Ensure that your user has access to read event data.

## Use of Managed Object References by the vSphere Provider

Unlike the vSphere client, many resources in the vSphere Terraform provider
take Managed Object IDs (or UUIDs when provided and practical) when referring to
placement parameters and upstream resources. This provides a stable interface
for providing necessary data to downstream resources, in addition to minimizing
the bugs that can arise from the flexibility in how an individual object's name
or inventory path can be supplied.

There are several data sources (such as
[`vsphere_datacenter`][tf-vsphere-datacenter],
[`vsphere_host`][tf-vsphere-host],
[`vsphere_resource_pool`][tf-vsphere-resource-pool],
[`vsphere_datastore`][tf-vsphere-datastore], and
[`vsphere_network`][tf-vsphere-network]) that assist with searching for a
specific resource in Terraform. For usage details on a specific data source,
look for the appropriate link in the sidebar. In addition, most vSphere
resources in Terraform supply the managed object ID (or UUID, when it makes
more sense) as the `id` attribute, which can be supplied to downstream
resources that should depend on the parent.

[tf-vsphere-host]: /docs/providers/vsphere/d/host.html

### Locating Managed Object IDs

There are certain points in time that you may need to locate the managed object
ID of a specific vSphere resource yourself. A couple of methods are documented
below.

#### Via `govc`

[govc][docs-govc] is an vSphere CLI built on [govmomi][docs-govmomi], the
vSphere Go SDK. It has a robust inventory browser command that can also be used
to list managed object IDs.

To get all the necessary data in a single output, use `govc ls -l -i PATH`.
Sample output is below:

```
$ govc ls -l -i /dc1/vm
VirtualMachine:vm-123 /dc1/vm/foobar
Folder:group-v234 /dc1/vm/subfolder
```

To do a reverse search, supply the `-L` switch:

```
$ govc ls -i -l -L VirtualMachine:vm-123
VirtualMachine:vm-123 /dc1/vm/foobar
```

For details on setting up govc, see the [homepage][docs-govc].

[docs-govc]: https://github.com/vmware/govmomi/tree/master/govc
[docs-govmomi]: https://github.com/vmware/govmomi


#### Via the vSphere Managed Object Browser (MOB)

The Managed Object Browser (MOB) allows one to browse the entire vSphere
inventory as it's presented to the API. It's normally accessed via
`https://VSPHERE_SERVER/mob`. For more information, see
[here][vsphere-docs-using-mob].

[vsphere-docs-using-mob]: https://code.vmware.com/doc/preview?id=4205#/doc/PG_Appx_Using_MOB.21.2.html#994699

~> **NOTE:** The MOB also offers API method invocation capabilities, and for
security reasons should be used sparingly. Modern vSphere installations may
have the MOB disabled by default, at the very least on ESXi systems. For more
information on current security best practices related to the MOB on ESXi,
click [here][vsphere-docs-esxi-mob].

[vsphere-docs-esxi-mob]: https://docs.vmware.com/en/VMware-vSphere/6.5/com.vmware.vsphere.security.doc/GUID-0EF83EA7-277C-400B-B697-04BDC9173EA3.html

## Bug Reports and Contributing

For more information how how to submit bug reports, feature requests, or
details on how to make your own contributions to the provider, see the vSphere
provider [project page][tf-vsphere-project-page].

[tf-vsphere-project-page]: https://github.com/hashicorp/terraform-provider-vsphere


