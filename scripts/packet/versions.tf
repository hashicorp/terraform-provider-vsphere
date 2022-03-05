terraform {
  required_providers {
    local = {
      source = "hashicorp/local"
    }
    metal = {
      source = "equinix/metal"
    }
    vsphere = {
      source = "hashicorp/vsphere"
      version = "2.0.2"
    }
  }
  required_version = ">= 0.13"
}
