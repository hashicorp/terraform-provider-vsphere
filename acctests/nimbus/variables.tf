variable "vcenter_username" {
  description = "Username used to authenticate against the vCenter Server"
  type = string
}

variable "vcenter_password" {
  description = "Password used to authenticate against the vCenter Server"
  type = string
}

variable "vcenter_server" {
  description = "FQDN or IP Address of the vCenter Server"
  type = string
}

variable "hosts" {
  type = list(object({
    hostname = string
    password = string
    username = string
  }))
}