# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    local = {
      source = "hashicorp/local"
    }
    vsphere = {
      source = "hashicorp/vsphere"
    }
    time = {
      source = "hashicorp/time"
    }
    null = {
      source = "hashicorp/null"
    }
    external = {
      source = "hashicorp/external"
    }
  }
  required_version = ">= 0.13"
}
