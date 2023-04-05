# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    local = {
      source = "hashicorp/local"
    }
    equinix = {
      source = "equinix/equinix"
    }
    time = {
      source = "hashicorp/time"
    }
  }
  required_version = ">= 0.13"
}
