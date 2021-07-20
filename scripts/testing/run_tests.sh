# You will likely need to 'export VSPHERE_PERSIST_SESSION=true' if you are running all tests as the API will start blocking new connection requests

cd ../../
for x in $(cat ./tests.txt); do TESTARGS="-count 1 -run $x" make testacc 2>&1 | tee ./scripts/testing/test_results/$x;done 
