generate_jobspec() {
    # The first argument is a JSON string
    local json_input="$1"

    # Verify jq installation and set absolute path
    local jq_path=$(command -v jq)
    if [[ -z "$jq_path" ]]; then
        echo "jq is required but not installed or not found in PATH."
        return 1  # Exit the function with an error status
    fi

    # Extract values using jq
    local engine=$("$jq_path" -r ".Engine" <<< "$json_input")
    local storage_sources=$("$jq_path" -r ".StorageSources" <<< "$json_input")
    local publishers=$("$jq_path" -r ".Publishers" <<< "$json_input")
    local count=$("$jq_path" -r ".Count" <<< "$json_input")
    local ROOT=$(git rev-parse --show-toplevel)
    local job_path="$ROOT/test/jobs"  # Changed variable name from PATH to job_path to avoid conflicts

    # Set the Name field for the jobspec
    local name="${engine} Job With StorageSource:${storage_sources} and Publisher:${publishers}"
    
    # Initialize JSON with base fields
    local jobspec_json=$("$jq_path" -n \
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
    local engine_json=$("$jq_path" -c ".[$engine_index].Engine" "$job_path/templates/engines.json")

    # Add engine JSON to the jobspec and include task name "main"
    jobspec_json=$(echo "$jobspec_json" | "$jq_path" --argjson engine "$engine_json" \
        '.Tasks[0] = {"Name": "main", "Engine": $engine}')

    # Handle input sources
    if [[ -n "$storage_sources" ]]; then
        local input_sources_json=$("$jq_path" -c ".[] | select(.InputSources[].Source.Type == \"$storage_sources\") | .InputSources" "$job_path/templates/input_sources.json")
        jobspec_json=$(echo "$jobspec_json" | "$jq_path" --argjson inputs "$input_sources_json" '.Tasks[0].InputSources = $inputs')
    fi

    # Handle publishers
    if [[ -n "$publishers" ]]; then
        local publisher_json=$("$jq_path" -c ".[] | select(.Publisher.Type == \"$publishers\")" "$job_path/templates/publishers.json")
        jobspec_json=$(echo "$jobspec_json" | "$jq_path" --argjson pub "$publisher_json.Publisher" '.Tasks[0].Publisher = $pub')
        jobspec_json=$(echo "$jobspec_json" | "$jq_path" --argjson paths "$publisher_json.ResultPaths" '.Tasks[0].ResultPaths = $paths')
    fi

    # Define filename based on parameters
    local filename="Engine${engine}_StorageSources${storage_sources}_Publishers${publishers}_Count${count}.json"

    # Output the complete jobspec JSON to a file
    echo "$jobspec_json" | "$jq_path" . > "$job_path/specs/$filename"
    echo "$filename"
}

# Example usage with JSON string argument:
filename=$(generate_jobspec '{"Engine":"docker", "StorageSources":"", "Publishers":"", "Count":2}')
echo "Generated jobspec file: $filename"
