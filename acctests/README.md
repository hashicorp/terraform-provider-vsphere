# Acceptance Tests
Acceptance testing is undergoing technical debt and streamlining. The current process recommends hosting on Equinix Metal, but the foundation of testing will hopefully evolve around nested ESXi VMs, ideally could be used to scale and parallelize tests.

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
Terraform needs to be run in two stages. First, to provision the Equinix infrastructure and second to create the shared vSphere resources upon which most acceptance tests rely.

```
$ cd acctests/equinix
$ terraform init
$ terraform apply
```
This process will take a significant amount of time, as a lot of data is being uploaded to servers.

## Provision vSphere
A file with several environment variables called `devrc` must be created within `acctests/equinix` path. You must source that file and then switch to the `acctests/vsphere` path.

```
$ source devrc
$ cd ../vsphere
$ terraform init
$ terraform apply
```
This will create a few resources expected in testing, as well as set a few default values for testing hardcoded in devrc.tpl. The set of variables were exported to another devrc, source that now.
```
$ source devrc
```

This set of environment variables and setup should allow you to run most of the acceptance tests, current failing ones or ones skipped due to missing or misconfigured environment variables will be addressed ASAP (unfortunately there had been a large amount of technical debt built up around testing).

### Please Note
As for writing a few manual steps had to be taken to privately network the nested ESXis and add them to vSphere. You will need to let the first apply fail, then perform the steps and re-apply.

1. Delete vmk1 and manually recreate it attached to the new vSwitch (which runs on vmnic1), give it an IP on the private subnet
2. Visit the physical ESXi web UI (likely the vCenter IP - 1) and power off the vcsa VM
3. Attach vmnet to the vcsa VM
4. Power on the vcsa VM
5. Visit the vCenter IP again, but this time port :5480
6. In the networking tab give it a valid IP on the private subnet the ESXi VMs run on
7. In vCenter manually create a snapshot of the template VM (this should be easy to capture in config going forward)

## Running tests
Tests can be ran via regexp pattern, generally advisable to run them individually or by resource type. It's common practice to have resource tests contain an underscore in their name to make the whole suite of tests for the resource run.

```
$ make testacc TESTARGS="-run=TestAccResourceVSphereVirtualMachine_ -count=1"
```

`count=1` is just a Golang trick to bust the testcache.

Some tests may leave a few resources behind causing a subsequent test in the suite to complain about a name already existing, generally you can just manually delete it from the vSphere UI to get the next test to run.