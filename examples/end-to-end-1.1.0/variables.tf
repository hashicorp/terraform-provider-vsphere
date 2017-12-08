// The datacenter the resources will be created in.
variable "datacenter" {
  type = "string"
}

// The hosts to use when creating virtual machines, mounting the datastore, and
// creating the distributed virtual switch. There should be 3 hosts defined
// here.
variable "esxi_hosts" {
  type = "list"
}

// The resource pool the virtual machines will be placed in.
variable "resource_pool" {
  type = "string"
}

// The name of the datastore to create.
variable "datastore_name" {
  type = "string"
}

// The DNS address of the NFS host to use for the datastore.
variable "nas_host" {
  type = "string"
}

// The export path of the NFS share to use for the datastore.
variable "nas_path" {
  type = "string"
}

// The name of the distributed virtual switch to create.
variable "switch_name" {
  type = "string"
}

// The name of the network interfaces to bind the DVS to on each host. These
// interfaces need to be available across all hosts defined in esxi_hosts.
variable "network_interfaces" {
  type = "list"
}

// The name of the port group to create.
variable "port_group_name" {
  type = "string"
}

// The VLAN ID of the port group. Setting this to 0 will untag the port group.
variable "port_group_vlan" {
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
