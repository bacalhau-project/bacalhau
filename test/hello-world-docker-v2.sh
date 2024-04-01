#!bin/bashtub

source bin/bacalhau-client.sh

testcase_can_run_docker_hello_world() {
    create_client "${CLUSTER}"
    # run the version command to initialize the repo and supress any errors around ClientID
    subject bacalhau version
    assert_equal 0 "${status}"

    # Run the job and get its ID
    subject bacalhau job run --id-only $ROOT/testdata/jobs/docker-hello.yaml
    assert_equal 0 "${status}"
    assert_equal '' "${stderr}"
    jobID="${stdout}"

    # describe the output of the job in json format
    subject bacalhau job describe --output=json "${jobID}"
    assert_equal 0 "${status}"
    assert_equal '' "${stderr}"

    # ensure there was only a single execution with expected stdout
    assert_equal 1 "$(echo "${stdout}" | jq '.Executions.Executions | length')"
    assert_match "Hello Bacalhau!\n" "$(echo "${stdout}" | jq '.Executions.Executions[0].RunOutput.Stdout')"
    assert_equal '' "${stderr}"
}