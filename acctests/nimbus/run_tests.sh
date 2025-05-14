#!/usr/bin/env bash
TF_ACC=1 go test -json -v ../../vsphere -timeout 360m 2>&1 | tee gotest.log | gotestfmt