cd ../../
for x in $(cat ./tests.txt); do TESTARGS="-count 1 -run $x" make testacc 2>&1 | tee ./scripts/testing/test_results/$x;done 
