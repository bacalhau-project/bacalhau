#!/bin/bash

generate_jobspec() {
    local json_input="$1"
    local engine=$(echo "$json_input" | jq -r ".Engine // \"default_engine\"")
    local storage_sources=$(echo "$json_input" | jq -r ".StorageSources // \"none\"")
    local publishers=$(echo "$json_input" | jq -r ".Publishers // \"none\"")
    local count=$(echo "$json_input" | jq -r ".Count // 1")

    local name="${engine} Job With StorageSource:${storage_sources} and Publisher:${publishers}"
    local jobspec_json=$(jq -n --arg name "$name" --argjson count "$count" '{
      "Name": $name,
      "Type": "batch",
      "Namespace": "default",
      "Count": ($count | tonumber),
      "Tasks": []
    }')

    # Debug output
    echo "Creating jobspec with: $jobspec_json"

    # Continue with your existing logic to handle engines, input sources, publishers...
}

# Call function with JSON
generate_jobspec '{"Engine":"Docker", "StorageSources":"", "Publishers":"", "Count":2}'
