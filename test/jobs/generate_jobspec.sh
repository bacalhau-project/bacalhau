generate_jobspec() {
  # The first argument is a JSON string
  local json_input="$1"

  # Extract and format input parameters
  local engine=$(echo "$json_input" | jq -r '.Engine')
  local storage_sources=$(echo "$json_input" | jq -r '.StorageSources')
  local publishers=$(echo "$json_input" | jq -r '.Publishers')
  local count=$(echo "$json_input" | jq -r '.Count')
  local ROOT=$(git rev-parse --show-toplevel)
  local job_path="$ROOT/test/jobs"  # Changed variable name from PATH to job_path

  # Define the jobspec base fields
  local jobspec_json=$(jq -n \
    --arg name "${engine} Job With StorageSource:${storage_sources} and Publisher:${publishers}" \
    --arg count "$count" \
    '{
      "Name": $name,
      "Type": "batch",
      "Namespace": "default",
      "Count": ($count | tonumber),
      "Tasks": []
    }')

  # Select the appropriate engine index
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

  # Fetch engine data
  local engine_json=$(jq -c ".[$engine_index].Engine" "$job_path/templates/engines.json")
  jobspec_json=$(echo "$jobspec_json" | jq --argjson engine "$engine_json" \
    '.Tasks[0] = {"Name": "main", "Engine": $engine}')

  # Process input sources and publishers
  if [[ -n "$storage_sources" ]]; then
    local input_sources_json=$(jq -c ".[] | select(.InputSources[].Source.Type == \"$storage_sources\") | .InputSources" "$job_path/templates/input_sources.json")
    jobspec_json=$(echo "$jobspec_json" | jq --argjson inputs "$input_sources_json" '.Tasks[0].InputSources = $inputs')
  fi
  if [[ -n "$publishers" ]]; then
    local publisher_json=$(jq -c ".[] | select(.Publisher.Type == \"$publishers\")" "$job_path/templates/publishers.json")
    jobspec_json=$(echo "$jobspec_json" | jq --argjson pub "$publisher_json.Publisher" '.Tasks[0].Publisher = $pub')
    jobspec_json=$(echo "$jobspec_json" | jq --argjson paths "$publisher_json.ResultPaths" '.Tasks[0].ResultPaths = $paths')
  fi

  # Construct the filename
  local filename="Engine${engine}_StorageSources${storage_sources}_Publishers${publishers}_Count${count}.json"

  # Save jobspec to file and echo the filename
  echo "$jobspec_json" | jq . > "$job_path/specs/$filename"
  echo "$job_path/specs/$filename"
}

# Example usage with capturing the filename:
filename=$(generate_jobspec '{"Engine":"Docker", "StorageSources":"ipfs", "Publishers":"", "Count":1}')
echo "Generated jobspec file: $filename"
