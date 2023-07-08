# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "vsphere_ovf_vm_template" "ubuntu" {
  name              = "ubuntu"
  disk_provisioning = "thin"
  resource_pool_id  = data.vsphere_compute_cluster.compute_cluster.resource_pool_id
  datastore_id      = data.vsphere_datastore.datastore2.id
  host_system_id    = data.vsphere_host.host1.id
  remote_ovf_url    = "https://cloud-images.ubuntu.com/releases/focal/release/ubuntu-20.04-server-cloudimg-amd64.ova"
  ovf_network_map = {
    "VM Network" = data.vsphere_network.public.id
  }
}
