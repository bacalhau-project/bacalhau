#!bin/bashtub

source bin/bacalhau-client.sh

testcase_can_get_results() {
    create_client "spawn"

    job_id=$(bacalhau job run --id-only $ROOT/testdata/jobs/docker-output.yaml)
    bacalhau get $job_id > /dev/null 2>&1
    subject cat job-*/output_custom/output.txt
    assert_equal 0 $status
    assert_match "15" $(echo $stdout)
    assert_equal '' $stderr
    extracted_id=$(echo $job_id | awk -F'-' '{print $1"-"$2}')
    rm -rf job-$extracted_id

}