# © Broadcom. All Rights Reserved.
# The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
# SPDX-License-Identifier: MPL-2.0

resource "local_file" "devrc" {
  content = templatefile("./devrc.tpl", {
    datacenter = vsphere_datacenter.dc.name
    cluster    = vsphere_compute_cluster.compute_cluster.name
    port_group = vsphere_host_port_group.pg.name
    trunk_nic  = vsphere_host_virtual_switch.switch.network_adapters[0]
  })
  filename = "./devrc"
}
