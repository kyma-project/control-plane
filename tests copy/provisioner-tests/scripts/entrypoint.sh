res=0

TEST_CASES=$1

if [ -z "$1" ]
  then
    TEST_CASES="Test_E2E_Gardener|TestShootUpgrade|TestRuntimeUpgrade"
  	echo "Test cases regexp not provided, running default test cases: $TEST_CASES"
fi

echo "Run Provisioner tests"
./provisioner.test -test.v -test.timeout 5h -test.run ${TEST_CASES} -test.parallel 3
res=$((res+$?))

exit ${res}