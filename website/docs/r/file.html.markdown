---
subcategory: "Storage"
layout: "vsphere"
page_title: "VMware vSphere: vsphere_file"
sidebar_current: "docs-vsphere-resource-storage-file"
description: |-
  Provides a VMware vSphere virtual machine file resource. This can be used to upload files (e.g. vmdk disks) from the Terraform host machine to a remote vSphere or copy fields within vSphere.
---

# vsphere\_file

The `vsphere_file` resource can be used to upload files (such as virtual disk
files) from the host machine that Terraform is running on to a target
datastore.  The resource can also be used to copy files between datastores, or
from one location to another on the same datastore.

Updates to destination parameters such as `datacenter`, `datastore`, or
`destination_file` will move the managed file a new destination based on the
values of the new settings.  If any source parameter is changed, such as
`source_datastore`, `source_datacenter` or `source_file`), the resource will be
re-created. Depending on if destination parameters are being changed as well,
this may result in the destination file either being overwritten or deleted at
the old location.

## Example Usages

### Uploading a file

```hcl
resource "vsphere_file" "ubuntu_disk_upload" {
  datacenter       = "my_datacenter"
  datastore        = "local"
  source_file      = "/home/ubuntu/my_disks/custom_ubuntu.vmdk"
  destination_file = "/my_path/disks/custom_ubuntu.vmdk"
}
```

### Copying a file

```hcl
resource "vsphere_file" "ubuntu_disk_copy" {
  source_datacenter = "my_datacenter"
  datacenter        = "my_datacenter"
  source_datastore  = "local"
  datastore         = "local"
  source_file       = "/my_path/disks/custom_ubuntu.vmdk"
  destination_file  = "/my_path/custom_ubuntu_id.vmdk"
}
```

## Argument Reference

If `source_datacenter` and `source_datastore` are not provided, the file
resource will upload the file from the host that Terraform is running on. If
either `source_datacenter` or `source_datastore` are provided, the resource
will copy from within specified locations in vSphere.

The following arguments are supported:

* `source_file` - (Required) The path to the file being uploaded from the
  Terraform host to vSphere or copied within vSphere. Forces a new resource if
  changed.
* `destination_file` - (Required) The path to where the file should be uploaded
  or copied to on vSphere.
* `source_datacenter` - (Optional) The name of a datacenter in which the file
  will be copied from. Forces a new resource if changed.
* `datacenter` - (Optional) The name of a datacenter in which the file will be
  uploaded to.
* `source_datastore` - (Optional) The name of the datastore in which file will
  be copied from. Forces a new resource if changed.
* `datastore` - (Required) The name of the datastore in which to upload the
  file to.
* `create_directories` - (Optional) Create directories in `destination_file`
  path parameter if any missing for copy operation. 
  
~> **NOTE:** Any directory created as part of the operation when
`create_directories` is enabled will not be deleted when the resource is
destroyed.
