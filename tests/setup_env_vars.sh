#!/usr/bin/env bash
export TF_ACC=1
export TF_LOG=INFO
export TF_VAR_STORAGE_POLICY=vSAN Default Storage Policy
export TF_VAR_VSPHERE_CLUSTER=acc-test-cluster
export TF_VAR_VSPHERE_DATACENTER=acc-test-dc
export TF_VAR_VSPHERE_ESXI1=
export TF_VAR_VSPHERE_ESXI2=
export TF_VAR_VSPHERE_ESXI3=
export TF_VAR_VSPHERE_ESXI4=
export TF_VAR_VSPHERE_NAS_HOST=
export TF_VAR_VSPHERE_NFS_DS_NAME="acc-test-nfs"
export TF_VAR_VSPHERE_PG_NAME='VM Network'
export TF_VAR_VSPHERE_RESOURCE_POOL=New Resource Pool
export VSPHERE_ALLOW_UNVERIFIED_SSL=true
export VSPHERE_PASSWORD=
export VSPHERE_SERVER=
export VSPHERE_USER=administrator@vsphere.local

