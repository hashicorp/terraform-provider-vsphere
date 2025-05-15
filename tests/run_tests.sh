#!/usr/bin/env bash
. setup_env_vars.sh
TF_ACC=1 go test -json -run "TestAccResourceVSphereComputeCluster_basic" -v ../vsphere -timeout 360m 2>&1 | tee gotest.log | gotestfmt