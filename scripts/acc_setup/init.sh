#!/bin/bash
set -e -u -o pipefail

if ! ping -c1 -W1 $TF_VAR_VSPHERE_SERVER; then
  sudo mount -o loop $TF_VAR_VCENTER_ISO /mnt
  cd init
  terraform init
  terraform apply --auto-approve
  rm terraform.tfstate*
  sudo umount /mnt
fi

