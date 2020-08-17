variable "VSPHERE_LICENSE" {
}

variable "VSPHERE_DATACENTER" {
}
  
variable "VSPHERE_CLUSTER" {                    
}

variable "VSPHERE_ESXI_TRUNK_NIC" {
}

variable "VSPHERE_RESOURCE_POOL" {
}

variable "VSPHERE_DVS_NAME" {
}
    
variable "VSPHERE_NFS_DS_NAME" {
}
  
variable "VSPHERE_PG_NAME" {
}
   
variable "VSPHERE_TEMPLATE" {}


provider "packet" {
  auth_token = "88jrny19JdBkQrpuGnKgWBaB5gjoYGtr"
}

provider "vsphere" {
  user                 = "administrator@vcenter.vspheretest.internal"
  password             = "Password123!"
  vsphere_server       = cidrhost("${packet_device.esxi1.network.0.address}/${packet_device.esxi1.network.0.cidr}",3)
  allow_unverified_ssl = true
}

locals {
  project_id = "9ca6b603-f95e-4e39-8e3e-fd3af0c223fa"
}

resource "packet_device" "esxi1" {
  hostname         = "esxi1.vspheretest.internal"
  plan             = "c3.medium.x86"
  facilities         = ["sjc1"]
  operating_system = "vmware_esxi_6_7"
  billing_cycle    = "hourly"
  project_id       = local.project_id
  network_type     = "hybrid"
}

resource "packet_device" "esxi2" {
  hostname         = "esxi2.vspheretest.internal"
  plan             = "c3.medium.x86"
  facilities         = ["sjc1"]
  operating_system = "vmware_esxi_6_7"
  billing_cycle    = "hourly"
  project_id       = local.project_id
  network_type     = "hybrid"
}

resource "packet_device" "storage1" {
  hostname         = "storage1.vspheretest.internal"
  plan             = "c1.small.x86"
  facilities         = ["sjc1"]
  operating_system = "ubuntu_20_04"
  billing_cycle    = "hourly"
  project_id       = local.project_id
  provisioner "remote-exec" {
    inline = [
      "mkdir /nfs",
      "apt-get update",
      "apt-get install nfs-common nfs-kernel-server -y",
      "echo \"/nfs *(rw,no_root_squash)\" > /etc/exports",
      "exportfs -a",
    ]
    connection {
      host = packet_device.storage1.network.0.address
      private_key = file("/home/hrich/.ssh/id_rsa")
    }
  }
}

data "packet_precreated_ip_block" "private" {
  facility         = "sjc1"
  project_id       = local.project_id
  address_family   = 4
  public           = false
}

data "packet_precreated_ip_block" "public" {
  facility         = "sjc1"
  project_id       = local.project_id
  address_family   = 4
  public           = true
}

resource "local_file" "vcsa_template" {
  content = templatefile("${path.cwd}/vcsa_deploy.json", {
    hostname       = packet_device.esxi1.network.0.address
    password       = packet_device.esxi1.root_password
    ip_address     = cidrhost("${packet_device.esxi1.network.0.address}/${packet_device.esxi1.network.0.cidr}",3)
    ip_prefix      = packet_device.esxi1.network.0.cidr
    gateway        = cidrhost("${packet_device.esxi1.network.0.address}/${packet_device.esxi1.network.0.cidr}",1)
    vcenter_fqdn   = "vcenter.vspheretest.internal"
    admin_password = "Password123!"
  })
  filename = "/tmp/vcsa.json"
  provisioner "local-exec" {
    command = "/mnt/vcsa-cli-installer/lin64/vcsa-deploy install --accept-eula --acknowledge-ceip --no-ssl-certificate-verification /tmp/vcsa.json"
  }
}

output "ip" {
  value = cidrhost("${packet_device.esxi1.network.0.address}/${packet_device.esxi1.network.0.cidr}",3)
}
