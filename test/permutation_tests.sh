#!/bin/bashtub

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
    # Check if the output is empty
    if [[ -z "$valid" ]]; then
        create_client "$CLUSTER"
        job_id=$(bacalhau job run --id-only $GENERATE_JOBSPECS/$filename)
        extracted_id=$(echo $job_id | awk -F'-' '{print $1"-"$2}')
        publishers=$(echo $json_input | jq -r '.Publishers')

        if [[ -z "$publishers" ]]; then
            # Handle the case when Publishers is empty
            subject bacalhau job logs $extracted_id
            continue
        else
            # Handle the case when Publishers is not empty
            bacalhau get $job_id > /dev/null 2>&1
            subject cat job-$extracted_id/output_custom/output.txt
        fi

        echo $GENERATE_JOBSPECS/$filename
        assert_equal 0 $status
        assert_match "hello" $(echo $stdout)
        assert_equal '' $stderr
        rm -rf job-$extracted_id
    else
        echo "Invalid jobspec file: $filename"
        echo "$valid"
    fi


  else
    echo "Requirements not met for configuration: $json_input"
  fi
done
rm -rf $GENERATE_JOBSPECS/*.json
}