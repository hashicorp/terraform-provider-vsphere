terraform {
  required_providers {
    local = {
      source = "hashicorp/local"
    }
    packet = {
      source = "terraform-providers/packet"
    }
    vsphere = {
      source = "hashicorp/vsphere"
    }
  }
  required_version = ">= 0.13"
}
