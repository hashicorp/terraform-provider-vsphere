# Notes

The .tf files within `base/` and `testrun/` are broken into long-lived vSphere resources and ephemeral resources provisioned each test run.

## Base

These resources are quite straight forward. The physical ESXI host running vSphere is added to inventory and other top level resources such as vSwitch/portgroup, datacenter, disks, and datastore.

## Testrun

These resources are what is needed to provide test coverage of the provider. The architecture relies on bringing up nested ESXI hosts which will be subject to testing, if anything goes wrong the VMs can be easily destroyed between runs to avoid having to rebuild the base ESXI/vSphere host which is quite lengthy. Other resources needed for testing have been virtualized such as an ubuntu VM serving as an NFS for storage tests.

The nested hosts run on a private network, this created a problem in retrieving their thumbprints needed for adding them to vSphere inventory. The workaround can be seen in `testrun/deploy-nested-esxi.tf` where we use a provisioner to SSH into the vSphere/top-level ESXI host which also resides on the private network, so it can retrieve the thumbprints via shell script.

### Please Note

As of writing the terraform configuration does not work. The foundation of everything was running tests against nested ESXI hosts (which prevented the lengthy build/teardown of a primary host). Unfortunately the OVAs were put behind a [Broadcom login](https://brcm.tech/flings). To restore this configuration, the images need to be downloaded and re-hosted somewhere else without a username/password login.

# GH Action

`.github/workflows/acceptance-tests.yaml` contains the configuration for automating the acceptance tests on GitHub as an Action. It's somewhat straightforward, it would use terraform to provision the `testrun/` ephemeral resources. The results would be be uploaded as a daily artifact to GitHub, the previous run would then be downloaded and compared against the current results. A small script would check if there were any new failures. In general Terraform providers operate against complex real world resources (datacenters) and often there is some level of "tests that are expected to fail", either because the tests themselves are poorly written, need updating, or the service being tested is just known to be flakey. The comparison script would try to avoid notification spam by only considering the testrun a failure if new tests had failed from the previous day.