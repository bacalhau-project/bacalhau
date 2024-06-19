#!bin/bashtub

source bin/bacalhau.sh

testcase_can_get_results() {
    # Assuming create_node is a function that takes two arguments separated by space, not comma
    create_node requester,compute

    job_id=$(bacalhau job run --id-only $ROOT/testdata/jobs/docker-output.yaml)
    bacalhau job get $job_id > /dev/null 2>&1
    subject cat job-*/output_custom/output.txt
    assert_equal 0 $status
    assert_match "15" $(echo $stdout)
    assert_equal '' $stderr
    rm -rf job-${job_id%%-*}
}