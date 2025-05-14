# Acceptance Tests

This directory contains a provider configuration and shell scripts that can configure a standard testing environment and trigger a test run.

## Prerequisites

You need to provision the bare infrastructure necessary to configure the testing environment.

This includes:

* 1 vCenter
* 4 ESX hosts with at least 2 network adapters
* 1 NFS share

## Configure Testing Environment

Create a `.tfvars` file and populate the variables declared in `variables.tf`.

Run `terraform apply` using the configuration provided in `main.tf` to prepare the environment for testing.

## Run Acceptance Tests

Set the missing values in `setup_env_vars.sh` and source the file.

Execute `run_tests.sh` to run the full test suite or add the `-run` parameter to the `go test` command to run a subset of tests.
