// example_virtual_machines creates a single virtual machine on each individual
// host.
resource "vsphere_virtual_machine" "example_virtual_machines" {
  count            = "${length(var.esxi_hosts)}"
  name             = "${var.virtual_machine_name_prefix}${count.index}"
  resource_pool_id = "${data.vsphere_resource_pool.example_resource_pool.id}"
  host_system_id   = "${data.vsphere_host.example_hosts.*.id[count.index]}"
  datastore_id     = "${data.vsphere_datastore.example_datastore.id}"

  num_cpus = 2
  memory   = 1024
  guest_id = "${data.vsphere_virtual_machine.example_template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.example_network.id}"
    adapter_type = "${data.vsphere_virtual_machine.example_template.network_interface_types[0]}"
  }

  disk {
    label = "disk0"
    size  = "${data.vsphere_virtual_machine.example_template.disks.0.size}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.example_template.id}"
    linked_clone  = true
  }

  vapp {
    properties {
      "guestinfo.coreos.config.data" = "${data.ignition_config.example_ignition_config.*.rendered[count.index]}"
    }
  }

  provisioner "file" {
    source      = "${path.module}/pkg/${var.server_binary_name}"
    destination = "${var.service_directory}/${var.server_binary_name}"

    connection {
      type        = "ssh"
      user        = "root"
      private_key = "${tls_private_key.example_provisioning_ssh_key.private_key_pem}"
    }
  }

  provisioner "remote-exec" {
    inline = [
      "chmod 755 ${var.service_directory}/${var.server_binary_name}",
      "chown ${var.service_user}:${var.service_user} ${var.service_directory}/${var.server_binary_name}",
      "systemctl enable ovf-example.service",
      "systemctl start ovf-example.service",
      "update-ssh-keys -u root -d coreos-ignition || /bin/true",
      "rm /root/.ssh/authorized_keys",
    ]

    connection {
      type        = "ssh"
      user        = "root"
      private_key = "${tls_private_key.example_provisioning_ssh_key.private_key_pem}"
    }
  }
}
