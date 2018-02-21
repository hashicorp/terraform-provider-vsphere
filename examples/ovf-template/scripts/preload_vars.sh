#!/usr/bin/env bash

# preload_vars.sh prompts you for environment variables that need to be set to
# ensure the build runs properly. Note that we don't do any sanity checking -
# if your build fails, re-run this script to attempt to add the correct vars.

vsphere_host=""
vsphere_username=""
vsphere_password=""
skip_ssl=""

read -r -p "Enter your vCenter/ESXi host: " vsphere_host
read -r -p "Enter your vCenter/ESXi username: " vsphere_username
read -r -s -p "Enter your vCenter/ESXi password: " vsphere_password
# newline is needed here because it is not echoed on slient input
echo
read -r -p "Do you want to skip SSL validation? (Please type \"yes\" if desired) " skip_ssl

export VSPHERE_SERVER="${vsphere_host}"
export GOVC_URL="${vsphere_host}"
export VSPHERE_USER="${vsphere_username}"
export GOVC_USERNAME="${vsphere_username}"
export VSPHERE_PASSWORD="${vsphere_password}"
export GOVC_PASSWORD="${vsphere_password}"
if [ "${skip_ssl}" == "yes" ]; then
  export VSPHERE_ALLOW_UNVERIFIED_SSL="true"
  export GOVC_INSECURE="true"
fi
