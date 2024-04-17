#!/bin/bash

# Function to read JSON arrays and objects
jq_installed=$(command -v jq)
if [ -z "$jq_installed" ]; then
  echo "jq is required but it's not installed. Please install jq to use this script."
  exit 1
fi

generate_jobspec() {
  # The first argument is a JSON string
  local json_input="$1"

  # Extract values from JSON string
  local engine=$(echo "$json_input" | jq -r '.Engine')
  local storage_sources=$(echo "$json_input" | jq -r '.StorageSources')
  local publishers=$(echo "$json_input" | jq -r '.Publishers')
  local count=$(echo "$json_input" | jq -r '.Count')

  # Set the Name field for the jobspec
  local name="${engine} Job With StorageSource:${storage_sources} and Publisher:${publishers}"
  
  # Initialize JSON with base fields
  local jobspec_json=$(jq -n \
    --arg name "$name" \
    --arg count "$count" \
    '{
      "Name": $name,
      "Type": "batch",
      "Namespace": "default",
      "Count": ($count | tonumber),
      "Tasks": []
    }')

  # Determine the engine index
  local engine_index
  if [[ -n "$storage_sources" && -n "$publishers" ]]; then
    engine_index=2
  elif [[ -n "$storage_sources" && -z "$publishers" ]]; then
    engine_index=0
  elif [[ -z "$storage_sources" && -n "$publishers" ]]; then
    engine_index=1
  else
    engine_index=3
  fi

  # Fetch the appropriate engine JSON
  local engine_json=$(jq -c ".[$engine_index].Engine" engines.json)

  # Add engine JSON to the jobspec and include task name "main"
  jobspec_json=$(echo "$jobspec_json" | jq --argjson engine "$engine_json" \
    '.Tasks[0] = {"Name": "main", "Engine": $engine}')

  # Handle input sources
  if [[ -n "$storage_sources" ]]; then
    local input_sources_json=$(jq -c ".[] | select(.InputSources[].Source.Type == \"$storage_sources\") | .InputSources" input_sources.json)
    jobspec_json=$(echo "$jobspec_json" | jq --argjson inputs "$input_sources_json" '.Tasks[0].InputSources = $inputs')
  fi

  # Handle publishers
  if [[ -n "$publishers" ]]; then
    local publisher_json=$(jq -c ".[] | select(.Publisher.Type == \"$publishers\")" publishers.json)
    jobspec_json=$(echo "$jobspec_json" | jq --argjson pub "$publisher_json.Publisher" '.Tasks[0].Publisher = $pub')
    jobspec_json=$(echo "$jobspec_json" | jq --argjson paths "$publisher_json.ResultPaths" '.Tasks[0].ResultPaths = $paths')
  fi

  # Define filename based on parameters
  local filename="Engine${engine}_StorageSources${storage_sources}_Publishers${publishers}_Count${count}.json"

  # Output the complete jobspec JSON to a file
  echo "$jobspec_json" | jq . > "$filename"
}

# Example usage with JSON string argument:
generate_jobspec '{"Engine":"Docker", "StorageSources":"ipfs", "Publishers":"", "Count":1}'
