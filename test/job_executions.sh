#!bin/bashtub

source bin/bacalhau.sh

testcase_can_describe_jobs() {
    # Assuming create_node is a function that takes two arguments separated by space, not comma
    create_node requester,compute
    job_id=$(bacalhau job run --id-only $ROOT/testdata/jobs/docker-hello.yaml)
    subject bacalhau job executions --output json $job_id
    assert_equal 0 $status
    assert_match $job_id $(jq -r '.[0].JobID' <<< "$stdout")
    assert_equal '' $stderr
}