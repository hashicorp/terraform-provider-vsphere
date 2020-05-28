resource "local_file" "vcsa_template" {
  content = templatefile("${path.cwd}/vcsa_deploy.json", {
    hostname       = var.host1
    password       = var.esxi_password
    ip_address     = var.vcenter_address
    ip_prefix      = var.vcenter_prefix
    gateway        = var.vcenter_gateway
    vcenter_fqdn   = var.vcenter_fqdn
    admin_password = "Password123!"
  })
  filename = "/tmp/vcsa.json"
  provisioner "local-exec" {
    command = "/mnt/vcsa-cli-installer/lin64/vcsa-deploy install --accept-eula --acknowledge-ceip --no-ssl-certificate-verification /tmp/vcsa.json"
  }
}

variable "esxi_password" {
  default = "changeme"
}

variable "host1" {
  default = "lab1.example.org"
}

variable "vcenter_address" {
  default = "192.168.10.31"
}

variable "vcenter_prefix" {
  default = "24"
}

variable "vcenter_gateway" {
  default = "192.168.10.1"
}

variable "vcenter_fqdn" {
  default = "vcenter.example.org"
}
