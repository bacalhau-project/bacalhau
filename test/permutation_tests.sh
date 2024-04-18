#!/bin/bash

# Source the script containing the job_requirements function
source bin/bacalhau-client.sh

# Path to the permutations.json file
PERMUTATIONS_FILE="jobs/permutations.json"

# Read the JSON file and loop over each permutation
jq -c '.[]' "$PERMUTATIONS_FILE" | while read -r permutation; do
  # Prepare JSON input directly using jq
  json_input=$(echo "$permutation" | jq '{Engine: .Engine, StorageSources: .StorageSources, Publishers: .Publishers, Count: .Count}')

  # Call the job_requirements function
#   echo "Testing configuration: $json_input"
  job_requirements "$json_input" "production"
  result=$?

  # Evaluate the result of the function call
  if [ $result -eq 0 ]; then
    # echo "Requirements met for configuration: $json_input"
    filename=$(generate_jobspec "$permutation")
    echo "$jobspec_json" | jq . > "$filename"
    # Ensure that all filesystem buffers are flushed
    sync
    
    echo "Vetting jobspec file: $filename"
    valid=$(cue vet -d "#Job" jobs/job-schema.cue $filename)
    # cue vet -d "#Job" jobs/job-schema.cue $filename
        # Check if the output is non-empty
    if [[ -z "$valid" ]]; then
      echo "Valid jobspec file: $filename"
    else
      echo "Invalid jobspec file: $filename"
      echo "$valid"
    fi

  else
    echo "Requirements not met for configuration: $json_input"
  fi
done
