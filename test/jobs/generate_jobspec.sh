generate_jobspec() {
    local json_input="$1"
    local jq_path=$(command -v jq)
    if [[ -z "$jq_path" ]]; then
        echo "jq is required but not installed or not found in PATH."
        return 1
    fi

    local ROOT=$(git rev-parse --show-toplevel)
    local job_path="$ROOT/test/jobs"
    local engines_path="$job_path/templates/engines.json"
    local input_sources_path="$job_path/templates/input_sources.json"
    local publishers_path="$job_path/templates/publishers.json"

    # Extract engine type, storage sources, publishers, and count from input JSON
    local engine_type=$(echo "$json_input" | "$jq_path" -r '.Engine')
    local storage_sources=$(echo "$json_input" | "$jq_path" -r '.StorageSources')
    local publishers=$(echo "$json_input" | "$jq_path" -r '.Publishers')
    local count=$(echo "$json_input" | "$jq_path" -r '.Count')
    
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

    # Construct the jobspec JSON directly in jq, reading necessary data from files using --slurpfile and --arg
    local jobspec_json=$(
      echo "$json_input" | "$jq_path" -r \
        --slurpfile engines "$engines_path" \
        --slurpfile input_sources "$input_sources_path" \
        --slurpfile publishers "$publishers_path" \
        --argjson engineIndex "$engine_index" \
        '
        . as $input |
        ($engines[0] | map(select(.Engine.Type == $input.Engine))[$engineIndex | tonumber].Engine) as $engine_data |
        ($input_sources[0] | map(select(.InputSources[].Source.Type == $input.StorageSources))[0].InputSources) as $inputs |
        ($publishers[0] | map(select(.Publisher.Type == $input.Publishers))[0].Publisher) as $publisher_data |
        ($publishers[0] | map(select(.Publisher.Type == $input.Publishers))[0].ResultPaths) as $result_paths |
        {
          "Name": ($engine_data.Type + " Job With StorageSource:" + $input.StorageSources + " and Publisher:" + $input.Publishers),
          "Type": "batch",
          "Namespace": "default",
          "Count": ($input.Count | tonumber),
          "Tasks": [
            {
              "Name": "main",
              "Engine": $engine_data
            } + 
            (if $inputs then { "InputSources": $inputs } else {} end) +
            (if $publisher_data then { "Publisher": $publisher_data } else {} end) +
            (if $result_paths then { "ResultPaths": $result_paths } else {} end) +
            { "Timeouts": { "ExecutionTimeout": 60 } }
          ]
        }
        '
    )

    local filename="Engine${engine_type}_StorageSources${storage_sources}_Publishers${publishers}_Count${count}.json"
    echo "$jobspec_json" > "$job_path/specs/$filename"
    echo "$filename"
}

# Example usage with JSON string argument:
filename=$(generate_jobspec '{"Engine":"docker", "StorageSources":"", "Publishers":"", "Count":2}')
echo "Generated jobspec file: $filename"
