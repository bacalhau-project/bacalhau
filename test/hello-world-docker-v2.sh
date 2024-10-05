#!bin/bashtub

source bin/bacalhau.sh

testcase_can_run_docker_hello_world() {
    # Assuming create_node is a function that takes two arguments separated by comma, not space
    create_node requester,compute

    subject ${BACALHAU} job run --follow $ROOT/testdata/jobs/docker-hello.yaml
    assert_equal 0 $status
    assert_match "Hello Bacalhau!" $(echo $stdout)
    assert_equal '' $stderr
}
