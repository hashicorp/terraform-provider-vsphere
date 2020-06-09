## Contributing to the provider

Thank you for your interest in contributing to the vSphere provider. We welcome your contributions. Here you'll find information to help you get started with provider development.

## Documentation

Our [provider development documentation](https://www.terraform.io/docs/extend/) provides a good start into developing an understanding of provider development. It's the best entry point if you are new to contributing to this provider.

To learn more about how to create issues and pull requests in this repository, and what happens after they are created, you may refer to the resources below:
- [Issue creation and lifecycle](ISSUES.md)
- [Pull Request creation and lifecycle](PULL_REQUESTS.md)


## Cloning the Project

First, you will want to clone the repository into your working directory:

```sh
git clone git@github.com:terraform-providers/terraform-provider-vsphere
```

## Running the Build

After the clone has been completed, you can enter the provider directory and
build the provider.

```sh
cd terraform-provider-vsphere
make build
```

## Installing the Local Plugin

After the build is complete, you can install the binary into your $GOPATH/bin folder with:

```sh
make install
```

# Developing the Provider

**NOTE:** Before you start work on a feature, please make sure to check the
[issue tracker][gh-issues] and existing [pull requests][gh-prs] to ensure that
work is not being duplicated. For further clarification, you can also ask in a
new issue.

[gh-issues]: https://github.com/terraform-providers/terraform-provider-vsphere/issues
[gh-prs]: https://github.com/terraform-providers/terraform-provider-vsphere/pulls

If you wish to work on the provider, you'll first need [Go][go-website]
installed on your machine (version 1.14+ is **required**). You'll also need to
correctly setup a [GOPATH][gopath], as well as adding `$GOPATH/bin` to your
`$PATH`.

[go-website]: https://golang.org/
[gopath]: http://golang.org/doc/code.html#GOPATH

See [Building the Provider](#building-the-provider) for details on building the provider.

# Testing the Provider

**NOTE:** Testing the vSphere provider is currently a complex operation as it
requires having a vCenter endpoint to test against, which should be hosting a
standard configuration for a vSphere cluster. Some of the tests will work
against ESXi, but YMMV.

## Configuring Environment Variables

Most of the tests in this provider require a comprehensive list of environment
variables to run. See the individual `*_test.go` files in the
[`vsphere/`](vsphere/) directory for more details. The next section also
describes how you can manage a configuration file of the test environment
variables.

### Using the `.tf-vsphere-devrc.mk` file

The [`tf-vsphere-devrc.mk.example`](../tf-vsphere-devrc.mk.example) file contains
an up-to-date list of environment variables required to run the acceptance
tests. Copy this to `$HOME/.tf-vsphere-devrc.mk` and change the permissions to
something more secure (ie: `chmod 600 $HOME/.tf-vsphere-devrc.mk`), and
configure the variables accordingly.

## Running the Acceptance Tests

After this is done, you can run the acceptance tests by running:

```sh
$ make testacc
```

If you want to run against a specific set of tests, run `make testacc` with the
`TESTARGS` parameter containing the run mask as per below:

```sh
make testacc TESTARGS="-run=TestAccVSphereVirtualMachine"
```

This following example would run all of the acceptance tests matching
`TestAccVSphereVirtualMachine`. Change this for the specific tests you want to
run.
