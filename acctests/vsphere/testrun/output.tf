# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "local_file" "devrc" {
  content = templatefile("./devrc.tpl", {
    esxi_host_2   = vsphere_host.nested-esxi[0].hostname
    esxi_host_3   = vsphere_host.nested-esxi[1].hostname
    esxi_host_4   = vsphere_host.nested-esxi[2].hostname
    vmfs_disk_0   = data.vsphere_vmfs_disks.available.disks[0]
    vmfs_disk_1   = data.vsphere_vmfs_disks.available.disks[1]
    vmfs_regexp   = data.vsphere_vmfs_disks.available.filter
    resource_pool = vsphere_resource_pool.pool.name
    nfs_ds_name   = vsphere_nas_datastore.ds.name
    template      = vsphere_virtual_machine.template.name
    test_ovf      = vsphere_virtual_machine.template.ovf_deploy[0].remote_ovf_url
    vm_name       = vsphere_virtual_machine.test-vm.name
    nas_host      = local.vsphere_nas_host
  })
  filename = "./devrc"
}
