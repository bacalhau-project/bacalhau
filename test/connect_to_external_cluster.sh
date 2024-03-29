#!bin/bashtub

source bin/bacalhau.sh

testcase_can_run_docker_hello_world_external_cluster() {
    export BACALHAU_NODE_CLIENTAPI_HOST=34.86.177.175
    export BACALHAU_NODE_CLIENTAPI_PORT=1234
    export BACALHAU_NODE_NETWORK_TYPE=nats
    export BACALHAU_NODE_NETWORK_ORCHESTRATORS=35.245.221.171:4222
    bacalhau job run --follow $ROOT/testdata/jobs/docker-hello.yaml
    subject bacalhau job run --follow $ROOT/testdata/jobs/docker-hello.yaml
    assert_equal 0 $status
    assert_match "Hello Bacalhau!" $(echo $stdout)
    assert_equal '' $stderr
}