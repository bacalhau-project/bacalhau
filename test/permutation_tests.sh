#!/bin/bash

# Source the script containing the job_requirements function
source bin/bacalhau-client.sh

testcase_can_run_permutation_tests() {
PERMUTATIONS_FILE="jobs/permutations.json"
GENERATE_JOBSPECS="jobs/specs"
# Read the JSON file and loop over each permutation
jq -c '.[]' "$PERMUTATIONS_FILE" | while read -r permutation; do
  # Prepare JSON input directly using jq
  json_input=$(echo "$permutation" | jq '{Engine: .Engine, StorageSources: .StorageSources, Publishers: .Publishers, Count: .Count}')


  job_requirements "$json_input" "production"
  result=$?

  # Evaluate the result of the function call
  if [ $result -eq 0 ]; then
    # echo "Requirements met for configuration: $json_input"
    filename=$(generate_jobspec "$permutation")
    # Ensure that all filesystem buffers are flushed
    # Check if cue command is available
    if command -v cue >/dev/null 2>&1; then
        # If cue is installed, run cue vet to check validity
        valid=$(cue vet -d "#Job" jobs/job-schema.cue $GENERATE_JOBSPECS/$filename > /dev/null)
    else
        # If cue is not installed, assume the file is valid
        valid="yes"
    fi
    echo $GENERATE_JOBSPECS/$filename
    # Check if the output is empty
    if [[ -z "$valid" ]]; then
        echo "Valid jobspec file: $filename"
        create_client "$CLUSTER"
        job_id=$(bacalhau job run --id-only $GENERATE_JOBSPECS/$filename)
        bacalhau get $job_id > /dev/null 2>&1
        subject cat job-*/output_custom/output.txt
        assert_equal 0 $status
        assert_match "hello" $(echo $stdout)
        assert_equal '' $stderr
        extracted_id=$(echo $job_id | awk -F'-' '{print $1"-"$2}')
        rm -rf job-$extracted_id
    else
        echo "Invalid jobspec file: $filename"
        echo "$valid"
    fi


  else
    echo "Requirements not met for configuration: $json_input"
  fi
done

}