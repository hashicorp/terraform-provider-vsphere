export TF_VAR_VSPHERE_REST_SESSION_PATH=$HOME/.govmomi/rest_sessions
export TF_VAR_VSPHERE_VIM_SESSION_PATH=$HOME/.govmomi/sessions

export TF_VAR_VSPHERE_NAS_HOST=${nas_host}
export TF_VAR_VSPHERE_ESXI1=${esxi_host_1}
export TF_VAR_VSPHERE_ESXI1_PW='${esxi_host_1_pw}'
export TF_VAR_VSPHERE_ESXI2=${esxi_host_2}
export TF_VAR_VSPHERE_ESXI2_PW='${esxi_host_2_pw}'
export TF_VAR_VSPHERE_ESXI3=${esxi_host_3}
export TF_VAR_VSPHERE_ESXI3_PW='${esxi_host_3_pw}'
export TF_VAR_VSPHERE_ESXI4=${esxi_host_4}
export TF_VAR_VSPHERE_ESXI4_PW='${esxi_host_4_pw}'
export VSPHERE_SERVER=${vsphere_host}

export TF_VAR_VSPHERE_VMFS_REGEXP='naa.'

export VSPHERE_USER="administrator@vcenter.vspheretest.internal"
export VSPHERE_PASSWORD="Password123!"
export VSPHERE_ALLOW_UNVERIFIED_SSL=true

export TF_VAR_VSPHERE_SERVER=$VSPHERE_SERVER
export TF_VAR_VSPHERE_USER=$VSPHERE_USER
export TF_VAR_VSPHERE_PASSWORD=$VSPHERE_PASSWORD
export TF_VAR_VSPHERE_ALLOW_UNVERIFIED_SSL=$VSPHERE_ALLOW_UNVERIFIED_SSL
export TF_VAR_VSPHERE_ESXI_TRUNK_NIC=vmnic1
export TF_VAR_VSPHERE_DATACENTER=hashidc
export TF_VAR_VSPHERE_CLUSTER=c1
export TF_VAR_VSPHERE_NFS_DS_NAME=nfs
export TF_VAR_VSPHERE_NFS_DS_NAME1=nfs-vol1
export TF_VAR_VSPHERE_NFS_DS_NAME2=nfs-vol2
export TF_VAR_VSPHERE_DVS_NAME=terraform-test-dvs
export TF_VAR_VSPHERE_PG_NAME='vmnet'
export TF_VAR_VSPHERE_RESOURCE_POOL=hashi-resource-pool
export TF_VAR_VSPHERE_NFS_PATH=/nfs
export TF_VAR_VSPHERE_NFS_PATH1=/nfs/ds1
export TF_VAR_VSPHERE_NFS_PATH2=/nfs/ds2
export TF_VAR_VSPHERE_ISO_DATASTORE=nfs
export TF_VAR_VSPHERE_ISO_FILE=fake.iso
export TF_VAR_VSPHERE_TEMPLATE=tfvsphere_template
export TF_VAR_VSPHERE_INIT_TYPE=thin
export TF_VAR_VSPHERE_ADAPTER_TYPE=lsiLogic
export TF_VAR_VSPHERE_DC_FOLDER=dc-folder
export TF_VAR_VSPHERE_DS_FOLDER=ds
export TF_VAR_VSPHERE_PERSIST_SESSION=true
export TF_VAR_VSPHERE_CLONED_VM_DISK_SIZE=20
export TF_VAR_VSPHERE_TEST_OVA="https://storage.googleapis.com/vsphere-acctest/TinyVM/TinyVM.ova"
export TF_VAR_VSPHERE_TEST_OVF="https://storage.googleapis.com/vsphere-acctest/TinyVM/TinyVM.ovf"
export TF_VAR_VSPHERE_CONTENT_LIBRARY_FILES="https://storage.googleapis.com/vsphere-acctest/TinyVM/TinyVM.ovf"

export TF_VAR_VSPHERE_HOST_NIC0=vmnic0
export TF_VAR_VSPHERE_HOST_NIC1=vmnic1


export TF_VAR_VSPHERE_VSWITCH_UPPER_VERSION="7.0.0"
export TF_VAR_VSPHERE_VSWITCH_LOWER_VERSION="6.5.0"
export TF_VAR_VSPHERE_DS_VMFS_NAME='ds-001'
export TF_VAR_VSPHERE_VM_V1_PATH='pxe-server'
export TF_VAR_VSPHERE_FOLDER_V0_PATH='Discovered virtual machine'
export TF_VAR_VSPHERE_ENTITY_PERMISSION_USER_GROUP="root"
