#!/bin/bash

# Source the script containing the job_requirements function
source bin/bacalhau-client.sh

# Path to the permutations.json file
PERMUTATIONS_FILE="jobs/permutations.json"

# Read the JSON file and loop over each permutation
jq -c '.[]' "$PERMUTATIONS_FILE" | while read -r permutation; do
  # Prepare JSON input directly using jq
  json_input=$(echo "$permutation" | jq '{Engine: .Engine, InputSources: .StorageSources, Publisher: .Publishers, count: .Count}')

  # Call the job_requirements function
#   echo "Testing configuration: $json_input"
  job_requirements "$json_input" "production"
  result=$?

  # Evaluate the result of the function call
  if [ $result -eq 0 ]; then
    # echo "Requirements met for configuration: $json_input"
    filename=$(generate_jobspec "$json_input")
    echo "Generated jobspec file: $filename"
  else
    echo "Requirements not met for configuration: $json_input"
  fi
done
