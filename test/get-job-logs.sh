#!bin/bashtub

source bin/bacalhau.sh

testcase_can_get_logs() {
    # Assuming create_node is a function that takes two arguments separated by space, not comma
    create_node requester,compute
    job_id=$(bacalhau job run --id-only $ROOT/testdata/jobs/docker-hello.yaml)
    subject bacalhau job logs $job_id
    assert_equal 0 $status
    assert_match "Hello Bacalhau!" $(echo $stdout | xargs)
    assert_equal '' $stderr
}