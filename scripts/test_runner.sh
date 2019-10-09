#!/bin/bash

main () {
  testList=$(TF_ACC=1 go test github.com/terraform-providers/terraform-provider-vsphere/vsphere | grep "\-\-\- FAIL" | awk '{ print $3 }')
  count=$(echo $testList | wc -w)
  for testName in $testList; do 
    runTest $testName
  done
}

runTest () {
  for try in {1..2}; do
    res=$(TF_ACC=1 go test github.com/terraform-providers/terraform-provider-vsphere/vsphere -v -count=1 -run="$1\$" -timeout 240m)
    if grep PASS <<< "$res" &> /dev/null; then
      break
    else
      revertAndWait
      sleep 30
    fi
  done
  echo "$res"
}


revertAndWait () {
  $GOPATH/src/github.com/terraform-providers/terraform-provider-vsphere/scripts/esxi_restore_snapshot.sh $VSPHERE_ESXI_SNAPSHOT
  sleep 30
  until curl -k "https://$VSPHERE_SERVER/rest/vcenter/datacenter" -i -m 10 2> /dev/null | grep "401 Unauthorized" &> /dev/null; do
    sleep 30
  done
}

main
