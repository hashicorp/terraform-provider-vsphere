# Acceptance Tests
Acceptance testing is undergoing revamping and streamlining. The current process recommends hosting on Equinix Metal.

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
Terraform needs to be run in a few stages. The first, to provision the Equinix infrastructure and deploy the main ESXi host.

```
$ cd acctests/equinix
$ terraform init
$ terraform apply
```
This process will take approximately 1.5h depending on network speeds. Once complete please source the `devrc` containing several key environment variables.

```
$ source devrc
```

## Provision vSphere
The vSphere infrastructure is provisioned in 2 steps, base and testrun. The base resources provision some basic cluster, networking and datastores, and adds the physical ESXi host into inventory.

### Base Step
Prior to applying, visit the physical ESXi web UI and find the unused boot disk, it will have a very long name like t10.ATA___XXX (assuming the use of Equinix c3.medium), and set `TF_VAR_VSPHERE_ESXI1_BOOT_DISK1` to this name and `TF_VAR_VSPHERE_ESXI1_BOOT_DISK1_SIZE` in GB (about half the overall disk size should be plenty, the nested ESXis will also need to use this datastore).

```
$ cd ../vsphere/base
$ terraform init
$ terraform apply
```

This will create another `devrc` file to source.

```
$ source devrc
```

A few manual steps need to be taken to privately network the nested ESXis setup in the testrun phase.

1. In vCenter, delete `vmk1` kernel adapter and manually recreate it attached to the new vSwitch created by Terraform  (`terraform-test` which runs on `vmnic1`), give it an IP on the private network (use the 2nd address, the nested ESXis will be setup to use address 3,4,5).
2. Visit the physical ESXi web UI and power off the vcsa VM
3. Attach `vmnet` to the vcsa VM
4. Power on the vcsa VM
5. Visit the vCenter IP again, but this time port `:5480`, signing in may fail the first time, but it is the vsphere username/password found in `acctests/equinix/devrc`
6. In the networking tab give it a valid IP on the private network (assuming 3 nested ESXIs will be created, use the 6th address).

### TestRun Step
This config can be destroyed between full test runs of the provider. It should cover cleaning up of things like the nested ESXis running in the wrong cluster, and cleaning up any leftover files in the NFS. Any lingering resources outside of those may need manual cleanup.

```
$ cd ../vsphere/testrun
$ terraform init
$ terraform apply
```

This will create a final `devrc` file to source.

```
$ source devrc
```

## Running tests
Now that you have all the required environment variables set, tests can be ran via regexp pattern. It is generally advisable to run them individually or by resource type. It's common practice to have resource tests contain an underscore in their name to make the whole suite of tests for the resource run.

```
$ make testacc TESTARGS="-run=TestAccResourceVSphereVirtualMachine_ -count=1"
```

`count=1` is just a Golang trick to bust the testcache.

# Nightly GitHub Action
The full suite of acceptance tests run rightly on GH Actions against ESXi/vSphere stood up on Equinix Metal. The `acctests/equinix` and `acctests/vsphere/base` should be long-lived, while `acctests/vsphere/testrun` is brought up and torn down between CI runs. As of writing the output simply pipes to a simple test summary script.

## Setup
From a local machine, please follow the instructions seen above up until `acctests/vsphere/base` is successfully applied. Through the GitHub UI, visit the repo settings and visit the "Environments" tab on the left. There should be an existing environment called `acctests` which is restricted to the `main` branch.

Configure the following as Environment variables:
```
TF_VAR_VSPHERE_CLUSTER
TF_VAR_VSPHERE_DATACENTER
TF_VAR_VSPHERE_ESXI1_BOOT_DISK1
TF_VAR_VSPHERE_ESXI1_BOOT_DISK1_SIZE
TF_VAR_VSPHERE_ESXI_TRUNK_NIC
TF_VAR_VSPHERE_PG_NAME
VSPHERE_ALLOW_UNVERIFIED_SSL
```

Configure the following as Environment **secrets**:
```
TF_VAR_VSPHERE_ESXI1
TF_VAR_VSPHERE_ESXI1_PW
TF_VAR_VSPHERE_LICENSE
TF_VAR_VSPHERE_PRIVATE_NETWORK
TF_VAR_VSPHERE_PUBLIC_NETWORK
VSPHERE_PASSWORD
VSPHERE_SERVER
VSPHERE_USER
SSH_PRIVATE_KEY
```

All of these variables and secrets should be exported to `acctests/equinix/devrc` and `acctests/vsphere/base/devrc`, or have been previously known to any applies (`TF_VAR_VSPHERE_LICENSE`). The `SSH_PRIVATE_KEY` secret should be the content of `acctests/equinix/gh-actions-ssh` (newline at the end included).

With these set, the nightly acceptances tests should run, normally the results of the test run should match what you can achieve locally (as in ran from your machine, but still pointed to vSphere hosted in Equinix), however there is some known flakiness that can derail a nightly:

* The remote nested ESXi OVA from `https://download3.vmware.com/software/vmw-tools/nested-esxi/` can fail to download during the setup steps of the GH action. Please simply re-run the action.
* Despite sweepers looking for it specifically, a common datastore called `nfs-vol2` can fail to unmount, causing widespread test failure. From the vSphere UI you will have to right-click and remove the datastore (it takes a few moments).
* Sometimes the API seems to lockup/crash or lockout the CI client resulting in widespread error 503s.

# Local Testing
The tests and required infrastructure have been heavily streamlined (believe it or not...) however its still expected they run Equinix for now, but it should be theoretically possible to run tests against local hardware (or perhaps totally virtualized). The main requirement is ESXi 7 with the following:

* A lot of memory (20GB or more), ESXi/vCenter will need a lot, and then there will be a few VMs at play for now that probably all need 0.5 to 1GB each.
* Probably 4 hard drives. The first hosts the vSphere and primary ESXi install (~70GB), a second one will serve as the datastore the NAS and 3 nested ESXi VMs run from (~100GB should be fine). The other 2 are needed as apart of the vmfs datastore tests, the size for these should not matter, make sure the override the regexp pattern for finding them `TF_VAR_VSPHERE_VMFS_REGEXP`.
* 2 NICs and 2 Networks. The topology of the test cluster is 1 IPv4 network on NIC0 (needs to be reachable by wherever Terraform runs from) and another IPv4 network on NIC1 (this network can be private to the caller/Terraform, but needs to be reachable by ESXi.. obviously). Two /29 subnets may look like this:
  * "Public" Network:
    1. Gateway
    2. Main ESXi host
    3. vCenter
    4. Ubuntu NAS created by `acctest/vsphere/testrun`
  * "Private" Network:
    1. Gateway
    2. Main ESXi host
    3. Nested ESXi host created by `acctest/vsphere/testrun`
    4. Nested ESXi host created by `acctest/vsphere/testrun`
    5. Nested ESXi host created by `acctest/vsphere/testrun`
    6. vCenter
