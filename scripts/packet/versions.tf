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
    }
  }
  required_version = ">= 0.13"
}
