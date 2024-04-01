#!bin/bashtub

source bin/bacalhau-client.sh

testcase_can_run_docker_hello_world() {
    create_client "$CLUSTER"
    subject bacalhau job run --follow $ROOT/testdata/jobs/docker-hello.yaml
    assert_equal 0 $status
    assert_match "Hello Bacalhau!" $(echo $stdout)
    assert_equal '' $stderr
}