#!/bin/bash
set -e -u -o pipefail

main () {
  eCode=0
  set +e
  testList=$(TF_ACC=1 go test github.com/hashicorp/terraform-provider-vsphere/vsphere | grep "\-\-\- FAIL" | awk '{ print $3 }' | grep TestAcc$1_)
  set -e
  for testName in $testList; do 
    runTest $testName  || eCode=1
  done
  exit $eCode
}

runTest () {
  eCode=0
  logfile="/tmp/tpv-build.log.$$"
  for try in {1..2}; do
    TF_ACC=1 go test github.com/hashicorp/terraform-provider-vsphere/vsphere -v -count=1 -run="$1\$" -timeout 240m 2>&1 | tee $logfile
    if grep PASS $logfile &> /dev/null; then
      eCode=0
      break
    else
      echo "test failed. reverting"
      revertAndWait
      sleep 30
      eCode=1
    fi
  done
  rm $logfile
  return $eCode
}


revertAndWait () {
  $GOPATH/src/github.com/hashicorp/terraform-provider-vsphere/scripts/esxi_restore_snapshot.sh $VSPHERE_ESXI_SNAPSHOT
  sleep 30
  until curl -k "https://$VSPHERE_SERVER/rest/vcenter/datacenter" -i -m 10 2> /dev/null | grep "401 Unauthorized" &> /dev/null; do
    echo -n . ;sleep 30
  done
  echo ' Done!'
}

main $1
