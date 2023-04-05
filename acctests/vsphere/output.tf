# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "local_sensitive_file" "devrc" {
  content = templatefile("./devrc.tpl", {
    esxi_host_2   = vsphere_host.nested-esxi[0].hostname
    esxi_host_3   = vsphere_host.nested-esxi[1].hostname
    esxi_host_4   = vsphere_host.nested-esxi[2].hostname
    datacenter    = vsphere_datacenter.dc.name
    cluster       = vsphere_compute_cluster.compute_cluster.name
    port_group    = vsphere_host_port_group.pg.name
    vmfs_disk_0   = data.vsphere_vmfs_disks.available.disks[0]
    vmfs_disk_1   = data.vsphere_vmfs_disks.available.disks[1]
    vmfs_regexp   = data.vsphere_vmfs_disks.available.filter
    resource_pool = vsphere_resource_pool.pool.name
    nfs_ds_name   = vsphere_nas_datastore.ds.name
    template      = vsphere_virtual_machine.template.name
    test_ovf      = vsphere_virtual_machine.template.ovf_deploy[0].remote_ovf_url
    vm_name       = vsphere_virtual_machine.pxe.name
    trunk_nic     = vsphere_host_virtual_switch.switch.network_adapters[0]
  })
  filename = "./devrc"
}
