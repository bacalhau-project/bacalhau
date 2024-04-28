#!bin/bashtub

source bin/bacalhau-client.sh

testcase_can_follow_job_logs() {
    create_client "$CLUSTER"
    # run the version command to initialize the repo and supress any errors around ClientID
    subject bacalhau version
    assert_equal 0 "${status}"
    subject bacalhau job run --follow $ROOT/testdata/jobs/wasm.yaml
    assert_equal 0 $status
    assert_match 'Hello, world!' $(echo $stdout | tail -n1)
    assert_equal '' $stderr
}
