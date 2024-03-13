## Contributing to the provider

Thank you for your interest in contributing to the vSphere provider. We welcome your contributions. Here you'll find information to help you get started with provider development.

## Documentation

Our [provider development documentation](https://www.terraform.io/docs/extend/) provides a good start into developing an understanding of provider development. It's the best entry point if you are new to contributing to this provider.

To learn more about how to create issues and pull requests in this repository, and what happens after they are created, you may refer to the resources below:

- [Issue Creation and Lifecycle](ISSUES.md)
- [Pull Request Creation and Lifecycle](PULL_REQUESTS.md)

## Cloning the Project

First, you will want to clone the repository into your working directory:

```shell
git clone git@github.com:hashicorp/terraform-provider-vsphere
```

## Running the Build

After the clone has been completed, you can enter the provider directory and build the provider.

```shell
cd terraform-provider-vsphere
make build
```

## Installing the Local Plugin

After the build is complete, you can install the binary into your `$GOPATH/bin` folder with:

```shell
make install
```

# Developing the Provider

**NOTE:** Before you start work on a feature, please make sure to check the [issue tracker][gh-issues] and existing [pull requests][gh-prs] to ensure that work is not being duplicated. For further clarification, you can also ask in a new issue.

[gh-issues]: https://github.com/hashicorp/terraform-provider-vsphere/issues
[gh-prs]: https://github.com/hashicorp/terraform-provider-vsphere/pulls

If you wish to work on the provider, you'll first need [Go][go-website] installed on your machine.

[go-website]: https://golang.org/
[gopath]: http://golang.org/doc/code.html#GOPATH

See [Building the Provider](#building-the-provider) for details on building the provider.

# Testing the Provider

Terraform providers tend to create, update, and destroy real resources to assert the provider is working as expected. This is called [Acceptance Testing](https://developer.hashicorp.com/terraform/plugin/sdkv2/testing/acceptance-tests). The vSphere provider's implementation is a bit more complex than the average provider, and creating a test environment that covers all possible hardware and settings combinations is a challenge. Effort has been put into streamlining the acceptance testing lab and instructions can be found in the acctests [README](/acctests/README.md).

# Maintaining the Changelog

In the future this should be automated, but between releases it's expected to add a SemVer entry at the top of the file with the following format.

```
## 2.4.0 (Unreleased)

FEATURES:
* `d/datasource_name`: Summary of the pull request. ([#1234](https://github.com/terraform-providers/terraform-provider-vsphere/pull/1234))
* `r/resource_name`: ...

BUG FIXES:
...

IMPROVEMENTS:
...

CHORES:
...
```
Generally the changes should fall into the categories listed above, when a resource or datasource is affected please follow the format seen above and link to the pull request (mind the brackets).

# Release the Provider

Releases will be performed by authorized HashiCorp, VMware, or community contributors. The release process is automated via GitHub Actions and is triggered by pushing a tag. To perform a release, please visit the Changelog and replace `(Unreleased)` with the current date (see the Changelog for the format).

Make sure to have pulled all the latest code changes to the main branch on your local machine (especially if the Changelog was edited via GitHub.com). When ready, create and push an annotated tag with the correct version number.
```
$ git tag -a v1.2.3 -m "v1.2.3"
$ git push --tag
```

The process should not require any additional actions. See the release workflow for details. Generally speaking the binaries should be built, signed, and uploaded in a matter of minutes. After which it can take up to an hour for the new release to be picked up by the Terraform Registry. If anything appears to have gone wrong, contact HashiCorp.