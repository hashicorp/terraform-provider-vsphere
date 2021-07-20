cd ../../
git grep -h '^func TestAcc.*'| sed -e 's/func \(TestAcc.*\)(.*/\1/' > ./scripts/testing/tests.txt