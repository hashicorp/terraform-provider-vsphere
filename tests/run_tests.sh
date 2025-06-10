#!/usr/bin/env bash
. setup_env_vars.sh
go install github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt@latest
TF_ACC=1 go test -json -v ../vsphere -timeout 360m 2>&1 | tee gotest.log | gotestfmt
