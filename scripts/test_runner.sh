TF_ACC=1 go test github.com/terraform-providers/terraform-provider-vsphere/vsphere | grep "\-\-\- FAIL" | awk '{ print $3 }' > /tmp/testlist
count=`wc -l /tmp/testlist | awk '{ print $1 }'`
for i in {1..2}; do
  for i in `cat /tmp/testlist`; do 
    res=`TF_ACC=1 go test github.com/terraform-providers/terraform-provider-vsphere/vsphere -v -count=1 -run="$i\$" -timeout 240m`; 
    echo $res | tee -a /tmp/testlog; 
    if echo $res | grep FAIL &> /dev/null; then
      echo Test failed. Reverting environment to clean state.
      $GOPATH/src/github.com/terraform-providers/terraform-provider-vsphere/scripts/esxi_restore_snapshot.sh $VSPHERE_ESXI_SNAPSHOT
      sleep 30
      until curl -k "https://$VSPHERE_SERVER/rest/vcenter/datacenter" -i -m 10 2> /dev/null | grep "401 Unauthorized" &> /dev/null; do
        echo vCenter not available yet. Waiting 30 seconds and trying again.
        sleep 30
      done
      sleep 30
    fi
  done
  grep "\-\-\- FAIL" /tmp/testlog | awk '{ print $3 }' > /tmp/testlist
  oldcount=$count
  count=`wc -l /tmp/testlist | awk '{ print $1 }'`
  if [ $oldcount -eq $count ]; then
    echo No improvement since last run. Exiting.
    exit 1
  fi
  if [ $count -eq 0 ]; then
    echo Tests complete.
    exit 0
  fi
  echo $count tests failed:
  cat /tmp/testlog
  rm /tmp/testlog
done
