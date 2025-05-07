# © Broadcom. All Rights Reserved.
# The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
# SPDX-License-Identifier: MPL-2.0

data "vsphere_ovf_vm_template" "nested-esxi" {
  name              = "Nested-ESXi-7.0"
  disk_provisioning = "thin"
  resource_pool_id  = data.vsphere_compute_cluster.compute_cluster.resource_pool_id
  datastore_id      = data.vsphere_datastore.datastore2.id
  host_system_id    = data.vsphere_host.host1.id
  remote_ovf_url    = "https://download3.vmware.com/software/vmw-tools/nested-esxi/Nested_ESXi7.0u3_Appliance_Template_v1.ova"
  ovf_network_map = {
    "${var.VSPHERE_PG_NAME}" = data.vsphere_network.pg.id
    "VM Network"             = data.vsphere_network.pg.id # second NIC for testing
  }
}
