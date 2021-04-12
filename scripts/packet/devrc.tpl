export TF_VAR_VCSA_DEPLOY_PATH="/tmp/vcsa/vcsa-cli-installer/mac/vcsa-deploy"
export TF_VAR_PRIV_KEY='<your ssh key>'
export TF_VAR_VSPHERE_REST_SESSION_PATH=$HOME/.govmomi/rest_sessions
export TF_VAR_VSPHERE_VIM_SESSION_PATH=$HOME/.govmomi/sessions
export TF_VAR_VSPHERE_LICENSE=00000-00000-00000-00000-00000 
export TF_VAR_PACKET_PROJECT=00000000-0000-0000-0000-000000000000
export TF_VAR_PACKET_AUTH=00000000000000000000000000000000
#export TF_VAR_ESXI_VERSION="vmware_esxi_7_0"
export TF_VAR_ESXI_VERSION="vmware_esxi_6_7"

export TF_VAR_VSPHERE_NAS_HOST=${nas_host}
export TF_VAR_VSPHERE_ESXI1=${esxi_host_1}
export TF_VAR_VSPHERE_ESXI2=${esxi_host_2}
export VSPHERE_SERVER=${vsphere_host}

export TF_VAR_VSPHERE_VMFS_REGEXP='naa.6000c29[2b]'
export TF_VAR_VSPHERE_DS_VMFS_ESXI1_DISK0="naa.6000c29b4b217432a854822b1bc40502"
export TF_VAR_VSPHERE_DS_VMFS_ESXI1_DISK1="naa.6000c29f3dbf773c6c25cb830dfc201b"

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
export TF_VAR_REMOTE_OVA_URL="https://storage.googleapis.com/acctest-images/tfvsphere_template.ova"
export TF_VAR_VSPHERE_TEMPLATE=tfvsphere_template
export TF_VAR_VSPHERE_INIT_TYPE=thin
export TF_VAR_VSPHERE_ADAPTER_TYPE=lsiLogic
export TF_VAR_VSPHERE_DC_FOLDER=dc-folder
export TF_VAR_VSPHERE_DS_FOLDER=ds
export TF_VAR_VSPHERE_USE_LINKED_CLONE=true
export TF_VAR_VSPHERE_PERSIST_SESSION=true
export TF_VAR_VSPHERE_CLONED_VM_DISK_SIZE=20
export TF_VAR_VSPHERE_TEST_OVA="https://storage.googleapis.com/acctest-images/yVM.ova"
export TF_VAR_VSPHERE_TEST_OVF="https://storage.googleapis.com/acctest-images/yVM.ovf"
export TF_VAR_VSPHERE_CONTENT_LIBRARY_FILES="https://storage.googleapis.com/acctest-images/yVM.ovf"
export TF_VAR_REMOTE_OVF_URL=https://acctest-images.storage.googleapis.com/tfvsphere_template.ovf

export TF_VAR_VSPHERE_HOST_NIC0=vmnic0
export TF_VAR_VSPHERE_HOST_NIC1=vmnic1


export TF_VAR_VSPHERE_VSWITCH_UPPER_VERSION="7.0.0"
export TF_VAR_VSPHERE_VSWITCH_LOWER_VERSION="6.5.0"
export TF_VAR_VSPHERE_DS_VMFS_NAME='ds-001'
export TF_VAR_VSPHERE_VM_V1_PATH='pxe-server'
export TF_VAR_VSPHERE_FOLDER_V0_PATH='Discovered virtual machine'
export TF_VAR_VSPHERE_ENTITY_PERMISSION_USER_GROUP="root"
