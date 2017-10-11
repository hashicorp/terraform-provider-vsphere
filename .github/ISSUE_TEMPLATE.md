Hi there,

Thank you for opening an issue. Please note that we try to keep the Terraform
issue trackers reserved for bug reports and feature requests. For general usage
questions, please see: https://www.terraform.io/community.html.

### Terraform Version

Run `terraform -v` to show the version. If you are not running the latest
version of Terraform, you can try upgrading, depending on if your problem is
with the provider or with Terraform core itself.

### vSphere Provider Version

Run `terraform providers` to show the list of providers in use and locate the
vSphere provider in the output (normally will show something like
`provider.vsphere`). If a version is not reported here, more than likely you are
not using a version lock. In this case, you can find the version number in the
`.terraform` data directory, usually the `.terraform/plugins/ARCH/` directory,
where `ARCH` is your system architecture (ie: `linux_amd64`, `windows_amd64`,
`darwin_amd64`, etc). In this directory, there will be a cached vSphere plugin
named something similar to terraform-provider-vsphere_v0.4.1_x4. Here, `0.4.1`
is the version number.

If you are running an older version of the plugin than the most recent version
(check the
[CHANGELOG](https://github.com/terraform-providers/terraform-provider-vsphere/blob/master/CHANGELOG.md)),
first try upgrading to make sure that you issue still persists, as it may have
been fixed in a later release.

### Affected Resource(s)

Please list the resources as a list, for example:
- `vsphere_virtual_machine`
- `vsphere_distributed_port_group`

If this issue appears to affect multiple resources, it may be an issue with
Terraform's core, so please mention this.

### Terraform Configuration Files

```hcl
# Copy-paste your Terraform configurations here - for large Terraform configs,
# please use a service like Dropbox and share a link to the ZIP file. For
# security, you can also encrypt the files using our GPG public key.
```

### Debug Output

Please provide a link to a GitHub Gist containing the complete debug output:
https://www.terraform.io/docs/internals/debugging.html. Please do NOT paste the
debug output in the issue; just paste a link to the Gist.

### Panic Output

If Terraform produced a panic, please provide a link to a GitHub Gist containing
the output of the `crash.log`.

### Expected Behavior

What should have happened?

### Actual Behavior

What actually happened?

### Steps to Reproduce

Please list the steps required to reproduce the issue, for example:
1. `terraform apply`

### Important Factoids

Are there anything atypical about your infrastructure that we should know about
that could be causing an edge case or something not necessarily obvious? If so,
please state it here.

### References

Are there any other GitHub issues (open or closed) or Pull Requests that should
be linked here? For example:
- GH-1234
