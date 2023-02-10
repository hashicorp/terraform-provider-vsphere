#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

set -e -u -o pipefail

for i in `govc ls /ha-datacenter/vm`; do 
  govc snapshot.revert -vm $i $VSPHERE_ESXI_SNAPSHOT; 
done
