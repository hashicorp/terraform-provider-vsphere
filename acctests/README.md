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
During the Equinix provisioning, macOS users may get security issues when ovftool is attempting to run, to prevent this issue run the following command:

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
```

## Provision Equinix Infrastructure
Terraform needs to be run in two stages. First, to provision the Equinix infrastructure and second to create the shared resources upon which most acceptance tests rely.

```
$ cd acctests/equinix
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
A file with several environment variables called `devrc` must be created within `acctests/equinix` path. You must source that file and then switch to the `acctests/vsphere` path.
```
$ source devrc
$ cd ../vsphere
$ terraform init
$ terraform apply
```
Now all the infrastructure should be created, if you planned on running the cluster tests with vSAN, make sure to enable vSAN on the VMkernal adapters on the management network for ESXI hosts 3 and 4. This can be done through the vSphere client or PowerCLI.  An additional `devrc` file should have been created in the `acctests/vsphere` path. Source this file and you should have all the necessary environment variables for running tests.
```
$ source devrc
```
