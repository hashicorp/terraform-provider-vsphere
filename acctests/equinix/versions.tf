# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    local = {
      source = "hashicorp/local"
    }
    metal = {
      source = "equinix/metal"
    }
  }
  required_version = ">= 0.13"
}
