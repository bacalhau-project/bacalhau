#!bin/bashtub

source bin/bacalhau.sh

testcase_ranking_failures_are_printed() {
    create_node compute,orchestrator

    subject ${BACALHAU} job run $ROOT/testdata/jobs/custom-task-type.yaml
    assert_match 'does not support flibble' $(echo $stdout)
}
