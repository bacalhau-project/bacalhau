#!bin/bashtub

source bin/bacalhau-client.sh

testcase_can_run_docker_hello_world_external_cluster() {
    create_client "production"
    bacalhau job run --follow $ROOT/testdata/jobs/docker-hello.yaml
    subject bacalhau job run --follow $ROOT/testdata/jobs/docker-hello.yaml
    assert_equal 0 $status
    assert_match "Hello Bacalhau!" $(echo $stdout)
    assert_equal '' $stderr
}