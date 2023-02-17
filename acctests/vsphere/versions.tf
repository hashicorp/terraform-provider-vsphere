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
  }
  required_version = ">= 0.13"
}
