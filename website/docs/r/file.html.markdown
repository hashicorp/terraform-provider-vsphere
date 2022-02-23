---
subcategory: "Storage"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_file"
sidebar_current: "docs-vsphere-resource-storage-file"
description: |-
  Provides a VMware vSphere file resource. This can be used to upload files
  (e.g. .iso and .vmdk) from the Terraform host machine to a remote vSphere
  or copy files within vSphere.
---

# vsphere\_file

The `vsphere_file` resource can be used to upload files (such as ISOs and
virtual disk files) from the host machine that Terraform is running on to a
datastore.  The resource can also be used to copy files between datastores, or
from one location to another on the same datastore.

Updates to destination parameters such as `datacenter`, `datastore`, or
`destination_file` will move the managed file a new destination based on the
values of the new settings.  If any source parameter is changed, such as
`source_datastore`, `source_datacenter`, or `source_file`), the resource will
be re-created. Depending on if destination parameters are being changed,
this may result in the destination file either being overwritten or
deleted from the previous location.

## Example Usages

### Uploading a File

```hcl
resource "vsphere_file" "ubuntu_vmdk_upload" {
  datacenter         = "dc-01"
  datastore          = "datastore-01"
  source_file        = "/my/src/path/custom_ubuntu.vmdk"
  destination_file   = "/my/dst/path/custom_ubuntu.vmdk"
  create_directories = true
}
```

### Copying a File

```hcl
resource "vsphere_file" "ubuntu_copy" {
  source_datacenter  = "dc-01"
  datacenter         = "dc-01"
  source_datastore   = "datastore-01"
  datastore          = "datastore-01"
  source_file        = "/my/src/path/custom_ubuntu.vmdk"
  destination_file   = "/my/dst/path/custom_ubuntu.vmdk"
  create_directories = true
}
```

## Argument Reference

If `source_datacenter` and `source_datastore` are not provided, the file
resource will upload the file from the host that Terraform is running on. If
either `source_datacenter` or `source_datastore` are provided, the resource
will copy from within specified locations in vSphere.

The following arguments are supported:

* `source_file` - (Required) The path to the file being uploaded from the
  Terraform host to the vSphere environment or copied within vSphere
  environment. Forces a new resource if changed.
* `destination_file` - (Required) The path to where the file should be uploaded
  or copied to on the destination `datastore` in vSphere.
* `source_datacenter` - (Optional) The name of a datacenter from which the file
  will be copied. Forces a new resource if changed.
* `datacenter` - (Optional) The name of a datacenter to which the file will be
  uploaded.
* `source_datastore` - (Optional) The name of the datastore from which file will
  be copied. Forces a new resource if changed.
* `datastore` - (Required) The name of the datastore to which to upload the
  file.
* `create_directories` - (Optional) Create directories in `destination_file`
  path parameter on first apply if any are missing for copy operation.

~> **NOTE:** Any directory created as part of the `create_directories` argument
  will not be deleted when the resource is destroyed. New directories are not
  created if the `destination_file` path is changed in subsequent applies.
