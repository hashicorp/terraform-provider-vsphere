// The datacenter the resources will be created in.
variable "datacenter" {
  type = "string"
}

// The hosts to use when creating virtual machines. There should be 3 hosts
// defined here.
variable "esxi_hosts" {
  type = "list"
}

// The resource pool the virtual machines will be placed in.
variable "resource_pool" {
  type = "string"
}

// The name of the datastore to use.
variable "datastore_name" {
  type = "string"
}

// The name of the network to use.
variable "network_name" {
  type = "string"
}

// The name of the template to use when cloning.
variable "template_name" {
  type = "string"
}

// The name prefix of the virtual machines to create.
variable "virtual_machine_name_prefix" {
  type = "string"
}

// The domain name to set up each virtual machine as.
variable "virtual_machine_domain" {
  type = "string"
}

// The network address for the virtual machines, in the form of 10.0.0.0/24.
variable "virtual_machine_network_address" {
  type = "string"
}

// The last octect that serves as the start of the IP addresses for the virtual
// machines. Given the default value here of 100, if the network address is
// 10.0.0.0/24, the 3 virtual machines will be assigned addresses 10.0.0.100,
// 10.0.0.101, and 10.0.0.102.
variable "virtual_machine_ip_address_start" {
  default = "100"
}

// The default gateway for the network the virtual machines reside in.
variable "virtual_machine_gateway" {
  type = "string"
}

// The DNS servers for the network the virtual machines reside in.
variable "virtual_machine_dns_servers" {
  type = "list"
}

// The name of the server binary to upload and start on the servers. This
// should already be built and reside in the "pkg" subdirectory of the working
// Terraform configuration directory.
variable "server_binary_name" {
  type = "string"
}

// The name of the user to be used when running the service binary.
variable "service_user" {
  type    = "string"
  default = "ovf-example"
}

// The home directory of the service user and the location where the service
// binary resides.
variable "service_directory" {
  type    = "string"
  default = "/ovf-example"
}

// A list of SSH keys that will be pushed to the "core" user on each CoreOS
// virtual machine. This allows for the management of each host after
// provisioning.
variable "management_ssh_keys" {
  type = "list"
}
