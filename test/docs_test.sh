#!/bin/bash

DOCS_PATH="../docs/docs/setting-up/workload-onboarding"
YAML_FILES=($(find "$DOCS_PATH" -name "*.yaml"))

# Prepare commands.txt with the needed commands
rm -f commands.txt  # Remove the file if it exists to start fresh
for file in "${YAML_FILES[@]}"; do
    echo "bacalhau job run --id-only $file" >> commands.txt
done

# Execute commands in parallel, capture output
output=$(cat commands.txt | parallel --keep-order --tag)

# Sleep for 300 seconds before querying status
sleep 300

# Initialize JSON output
json_output="["  # Start the JSON array
first=1  # Flag to handle comma-separation in JSON

# Read each line from output
while IFS= read -r line; do
    if [[ "$line" =~ j- ]]; then
        filepath=$(echo "$line" | awk -F 'job run --id-only ' '{print $2}' | awk '{print $1}')
        job_id=$(echo "$line" | awk -F 'job run --id-only ' '{print $2}' | awk '{print $2}' | tr -d '[:space:]')

        # Generate md_path from the filepath
        dir_name=$(dirname "$filepath")
        md_path="$dir_name/index.md"  # Correctly form the md_path

        status=$(bacalhau job describe "$job_id" --output json | jq -r '.Job.State.StateType' 2>/dev/null || echo "Failed to retrieve status")
        exit_code=$(bacalhau describe "$job_id" --json | jq '.State.Executions[0].RunOutput.exitCode' 2>/dev/null || echo "Failed to retrieve exit code")

        if [ "$first" -eq 1 ]; then
            first=0
        else
            json_output+=","
        fi
        json_output+="{\"script_path\":\"$filepath\", \"job_id\":\"$job_id\", \"md_path\":\"$md_path\", \"status\":\"$status\", \"exit_code\":\"$exit_code\"}"
    fi
done <<< "$output"

json_output+="]"  # End the JSON array

results="$json_output"


# Loop through each job in the JSON output
echo "$results" | jq -c '.[]' | while IFS= read -r item; do
    job_status=$(echo "$item" | jq -r '.status')
    exit_code=$(echo "$item" | jq -r '.exit_code')
    md_path=$(echo "$item" | jq -r '.md_path')
    # Determine badge color and label based on status
    if [[ "$job_status" == "Completed" ]] && [[ "$exit_code" == "0" ]]; then
        badge_color="green"
        badge_label="Pass"
    else
        badge_color="red"
        badge_label="Fail"
    fi

    # Construct the badge URL
    badge_url="https://img.shields.io/badge/Test-$badge_label-$badge_color"

    # Check if the markdown file exists
    if [ -f "$md_path" ]; then
        # Use sed to replace the existing badge URL
        # This regex matches the part of the URL up to the "Test-" label, followed by any text up to the last hyphen, which is presumed to be the color
        sed -i '' -E "s|(https://img.shields.io/badge/Test-[^-]+-[^-]+)([^)]*\))|$badge_url\2|" "$md_path"
    else
        echo "Markdown file does not exist: $md_path"
    fi
done