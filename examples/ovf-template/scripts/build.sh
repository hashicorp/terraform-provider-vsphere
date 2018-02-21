#!/usr/bin/env bash

HTTP_HELLO_GO_PATH="github.com/vancluever/http-hello-go"

set -e

SRV_BIN="$(grep "server_binary_name" "${TF_DIR}/terraform.tfvars" | awk '{print $3}' | tr -d \'\"\' )"

mkdir -p "${TF_DIR}/pkg"
go get -d "${HTTP_HELLO_GO_PATH}"
CGO_ENABLED=0 go build -o "${TF_DIR}/pkg/${SRV_BIN}" "${HTTP_HELLO_GO_PATH}"
