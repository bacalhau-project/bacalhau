#!bin/bashtub

source bin/bacalhau-client.sh

testcase_can_get_logs() {
    create_client "$TEST_ENV"
    job_id=$(bacalhau job run --id-only $ROOT/testdata/jobs/docker-hello.yaml)
    subject bacalhau job logs $job_id
    assert_equal 0 $status
    assert_match "Hello Bacalhau!" $(echo $stdout | xargs)
    assert_equal '' $stderr
}