#!/usr/bin/env bash

set -e

TEMPLATE_URL="https://stable.release.core-os.net/amd64-usr/current/coreos_production_vmware_ova.ova"
TEMPLATE_DC="$(grep "datacenter" "${TF_DIR}/terraform.tfvars" | awk '{print $3}' | tr -d \'\"\' )"
TEMPLATE_DS="$(grep "datastore_name" "${TF_DIR}/terraform.tfvars" | awk '{print $3}' | tr -d \'\"\' )"
TEMPLATE_POOL="$(grep "resource_pool" "${TF_DIR}/terraform.tfvars" | awk '{print $3}' | tr -d \'\"\' )"
TEMPLATE_NAME="$(grep "template_name" "${TF_DIR}/terraform.tfvars" | awk '{print $3}' | tr -d \'\"\' )"

EXISTING_TEMPLATE="$(govc find -type m -name "${TEMPLATE_NAME}" | head -n 1)"
if [ -n "${EXISTING_TEMPLATE}" ]; then
  govc object.rename "${EXISTING_TEMPLATE}" "${TEMPLATE_NAME}_archived_$(date +%s)"
fi

govc import.ova -dc="${TEMPLATE_DC}" -ds="${TEMPLATE_DS}" -pool="${TEMPLATE_POOL}" -name="${TEMPLATE_NAME}" "${TEMPLATE_URL}"
govc snapshot.create -dc="${TEMPLATE_DC}" -vm="${TEMPLATE_NAME}" clone-root
govc vm.markastemplate -dc="${TEMPLATE_DC}" "${TEMPLATE_NAME}"
