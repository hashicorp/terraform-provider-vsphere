# Acceptance Tests

Here is the current process for deploying infrastructure that can be used for running acceptance tests. Currently only supported on Equinix Metal, the metal provider is currently set for EOL in July 2023, a new official process for running tests will need to be developed.

## Download

1. Log in to VMware Customer Connect.
2. Navigate to Products and Accounts > All Products.
3. Find VMware vSphere and click View Download Components.
4. Select a VMware vSphere version from the Select Version drop-down.
5. Select a version of VMware vCenter Server and click GO TO DOWNLOADS.
6. Download the vCenter Server appliance ISO image.

## Extract

Next you need to mount and extract the contents of the iso to a location on your disk.

### macOS Users
During phase 1 of the setup, macOS users may get security issues when ovftool is attempting to run, to prevent this issue please run:

```
$ sudo xattr -r -d com.apple.quarantine {extract_location}/vcsa/ovftool/mac/ovftool
```

## Prepare Environment Variable
Set the following environment variables
```
# the prefix is only required to prevent collisions with other users in your equinix project
export TF_VAR_LAB_PREFIX="your prefix"
# make sure to add a ssh pubkey to your equinix account, set this variable to the privkey
export TF_VAR_PRIV_KEY="/Users/you/.ssh/id_rsa"
# ID of the equinix project
export TF_VAR_PACKET_PROJECT="project id"
# create an auth token in equinix and set it here
export TF_VAR_PACKET_AUTH="auth token"
# your vsphere licence
export TF_VAR_VSPHERE_LICENSE="XXXXX-XXXXX-XXXXX-XXXXX-XXXXX"
# set this to the location of the extracted vsca installer for your OS
export TF_VAR_VCSA_DEPLOY_PATH="{extract_location}/vcsa-cli-installer/mac/vcsa-deploy"
# make sure the following variables are set, you can rename them but it's possible these values are hardcoded somewhere
# it is safest to leave them to these values
export TF_VAR_VSPHERE_ESXI_TRUNK_NIC=vmnic1
export TF_VAR_VSPHERE_DATACENTER=hashidc
export TF_VAR_VSPHERE_CLUSTER=c2
export TF_VAR_VSPHERE_NFS_DS_NAME=nfs
export TF_VAR_VSPHERE_DVS_NAME=terraform-test-dvs
export TF_VAR_VSPHERE_PG_NAME=vmnet
export TF_VAR_VSPHERE_RESOURCE_POOL=hashi-resource-pool
export TF_VAR_VSPHERE_TEMPLATE=tfvsphere_template
```

## Run Terraform Phase 1
Terraform must be ran in 2 applies, first for Equinix, and a second apply sets up the expected vSphere environment. The config has specific Equinix regions and server sizes set, it's best to leave these to their default values seen in `main.tf.phase1`

### Please Note
Due to a problem in the provider block configuration, the latest version of Terraform that can be used is v1.2 for now.

Also be aware that this config deploys 5 servers, 4 ESXI hosts and 1 NFS (which will be expensive). 2 of them are ESXI hosts used exclusively for some cluster tests, if you don't plan on running those, you could cut the related config from `phase1` and `phase2` to save some cost.

```
$ cp main.tf.phase1 main.tf
$ terraform init
$ terraform apply
```
This process will take a significant amount of time, as a lot of data is being uploaded to servers. The process likely completed successfully but errored with something like
```
Error 422: still bonded 
```
You should be able to just reapply
```
$ terraform apply
```

## Run Terraform Phase 2
Phase 1 should have created a file with more environment variables called `devrc`. You should source that file, and then re-source wherever you saved your previous environment variables (so the required secrets get picked up).
```
$ source devrc
$ source ~/.zshenv # or wherever you set them
$ cp main.tf.phase2 main.tf
$ terraform apply
```
There is an issue where a network is unavailable right away and some test VMs will fail to create, running a second apply should fix this.
```
$ terraform apply
```
Now all the infrastructure should be created, if you planned on running the cluster tests with vSAN, make sure to enable vSAN in VMKernal Adapters on the management network for ESXI hosts 3 and 4. This is must be done through the web console. You should have most of the necessary environment variables for acceptances tests by sourcing
```
$ source devrc
$ source ~/.zshenv
```
