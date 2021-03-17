#!/bin/bash
set -e -u -o pipefail

cat << EOF > /tmp/config.yml
version: 2
jobs:
EOF

tests=$(TF_ACC=1 go test github.com/hashicorp/terraform-provider-vsphere/vsphere | grep "\-\-\- FAIL" | awk '{ print $3 }')
categories=$(sed -e s/^TestAcc//g <<< "$tests" | sed -e s/_.*//g | sort -u)
cat << EOF >> /tmp/config.yml
  linters:
    docker: 
    - image: circleci/golang:1.16
    working_directory: /home/circleci/src/github.com/hashicorp/terraform-provider-vsphere
    steps:
    - checkout
    - run:
        name: "Move to GOPATH"
        command: |
          mkdir -p \$GOPATH/src/github.com/hashicorp/terraform-provider-vsphere
          mv /home/circleci/src/github.com/hashicorp/terraform-provider-vsphere/* \$GOPATH/src/github.com/hashicorp/terraform-provider-vsphere
    - run:
        name: "Get tfproviderlint"
        command: |
          go get -u github.com/bflad/tfproviderlint
    - run:
        no_output_timeout: 30m
        name: "Run tfproviderlint"
        command: |
          CGO_ENABLED=1 tfproviderlint -AT00{1..3} -R00{2..4} -S0{{01..12},{14..19}}  ./...
EOF

for category in $categories; do
	cat << EOF >> /tmp/config.yml
  test_acc_$category:
    docker: 
    - image: circleci/golang:1.16
    working_directory: /home/circleci/src/github.com/hashicorp/terraform-provider-vsphere
    steps:
    - checkout
    - run:
        name: "Move to GOPATH"
        command: |
          mkdir -p \$GOPATH/src/github.com/hashicorp/terraform-provider-vsphere
          mv /home/circleci/src/github.com/hashicorp/terraform-provider-vsphere/* \$GOPATH/src/github.com/hashicorp/terraform-provider-vsphere
    - add_ssh_keys:
        fingerprints:
          - "62:4d:8d:04:48:f7:0f:5a:63:da:de:a6:30:f4:b4:12"
    - run:
        name: "SSH Tunnel"
        command: |
          ssh \$TUNNEL_USER@\$TUNNEL_HOST -L 4430:vcenter.vsphere.hashicorptest.internal:443 -L 4431:esxi1.vsphere.hashicorptest.internal:443 -o StrictHostKeyChecking=no -f sleep 32400
    - run:
        name: "Get GOVC"
        command: |
          go get -u github.com/vmware/govmomi/govc
    - run:
        no_output_timeout: 30m
        name: "Run Acceptance Tests"
        command: |
          \$GOPATH/src/github.com/hashicorp/terraform-provider-vsphere/scripts/test_runner.sh $category

EOF
done

cat << EOF >> /tmp/config.yml
workflows:
  version: 2
  commit:
#    triggers:
#      - schedule:
#          cron: "0 3 * * *"
#          filters: 
#            branches:
#              only:
#                - master
    jobs:
      - linters
EOF

unset $lastCat
for category in $categories; do
  [ -z "$lastCat" ] && echo "      - test_acc_$category" >> /tmp/config.yml
  [ -z "$lastCat" ] || echo "      - test_acc_$category:" >> /tmp/config.yml
  [ -z "$lastCat" ] || echo "          requires:" >> /tmp/config.yml
  [ -z "$lastCat" ] || echo "            - test_acc_$lastCat" >> /tmp/config.yml
  lastCat=$category
done
