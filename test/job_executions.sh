#!bin/bashtub

source bin/bacalhau-client.sh

testcase_can_describe_jobs() {
    create_client "$CLUSTER"
    job_id=$(bacalhau job run --id-only $ROOT/testdata/jobs/docker-hello.yaml)
    subject bacalhau job executions --output json $job_id
    assert_equal 0 $status
    assert_match $job_id $(jq -r '.[0].JobID' <<< "$stdout")
    assert_equal '' $stderr
}