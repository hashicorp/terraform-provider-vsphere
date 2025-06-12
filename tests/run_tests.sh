#!/usr/bin/env bash
. setup_env_vars.sh
TF_ACC=1 go test -p=1 -json -v ../vsphere -timeout 360m 2>&1 | tee gotest.log | gotestfmt