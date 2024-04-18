source bin/bacalhau.sh

clean_repo_external() {
find "${BACALHAU_DIR}" -mindepth 1 \( ! -name 'config.yaml' ! -name 'token.txt' ! -name 'metadata.json' \) -exec rm -rf {} +
}

create_client() {
# Set 'spawn' as default value if $1 is not provided
environment=${1:-spawn}

case "$environment" in
  production)
    echo "Running in production environment"
    export BACALHAU_DIR=$ROOT/test/clusters/production
    ;;
  staging)
    echo "Running in staging environment"
    export BACALHAU_DIR=$ROOT/test/clusters/staging
    ;;
  development)
    echo "Running in development environment"
    export BACALHAU_DIR=$ROOT/test/clusters/development
    ;;
  local)
    echo "Running in local environment"
    export BACALHAU_DIR=$ROOT/test/clusters/local
    # uses an existing cluster, doesn't delete anything after test runs
    ;;
  spawn)
    echo "Spawning a new environment"
    create_node requester,compute
    ;;
  private)
    echo "Running Against Private Cluster"
    export BACALHAU_DIR=$BACALHAU_CUSTOM_CONFIG
  ;;
  *)
    echo "Unknown environment: $environment"
    ;;
esac
}

job_requirements() {
  local json_input="$1"
  local environment="$2"
  local ROOT=$(git rev-parse --show-toplevel)
  local Engine=$(echo "$json_input" | jq -r '.Engine | ascii_downcase')
  local StorageSources=$(echo "$json_input" | jq -r '.StorageSources | ascii_downcase')
  local Publishers=$(echo "$json_input" | jq -r '.Publishers | ascii_downcase')
  local count=$(echo "$json_input" | jq -r '.Count')

  # echo "Testing with Engine=$Engine, StorageSources=$StorageSources, Publishers=$Publishers, count=$count"

  case "$environment" in
    production)
      export BACALHAU_DIR="$ROOT/test/clusters/production"
      ;;
    staging)
      export BACALHAU_DIR="$ROOT/test/clusters/staging"
      ;;
    development)
      export BACALHAU_DIR="$ROOT/test/clusters/development"
      ;;
    local)
      export BACALHAU_DIR="$ROOT/test/clusters/local"
      ;;
    private)
      export BACALHAU_DIR="${BACALHAU_CUSTOM_CONFIG:-$ROOT/test/clusters/private}"
      ;;
    *)
      echo "Unknown environment: $environment" >&2
      return 1
      ;;
  esac

  local metadata_path="$BACALHAU_DIR/metadata.json"
  if [[ ! -s "$metadata_path" ]]; then
    echo "No metadata found at $metadata_path"
    return 1
  fi

  local metadata=$(cat "$metadata_path")
  local match_count=0
  local total_nodes=$(echo "$metadata" | jq '. | length')

  for (( i = 0; i < total_nodes; i++ )); do
    local engine_match=$(echo "$metadata" | jq -r ".[$i].ComputeNodeInfo.ExecutionEngines[] | ascii_downcase | select(. == \"$Engine\")")
    local publisher_match=$(echo "$metadata" | jq -r ".[$i].ComputeNodeInfo.Publishers[] | ascii_downcase | select(. == \"$Publishers\")")
    local source_match=$(echo "$metadata" | jq -r ".[$i].ComputeNodeInfo.StorageSources[] | ascii_downcase | select(. == \"$StorageSources\")")

    # echo "Node $i: Engine match=$engine_match, Publishers match=$publisher_match, Input Source match=$source_match"

    if [[ -n "$engine_match" ]] && 
       ([[ -z "$Publishers" ]] || [[ -n "$publisher_match" ]]) && 
       ([[ -z "$StorageSources" ]] || [[ -n "$source_match" ]]); then
      ((match_count++))
    fi
  done

  # echo "Total matches found: $match_count"
  if [[ "$match_count" -ge "$count" ]]; then
    return 0
  else
    return 1
  fi
}

generate_jobspec() {
    local json_input="$1"

    # Verify jq installation and set absolute path
    local jq_path=$(command -v jq)
    if [[ -z "$jq_path" ]]; then
        echo "jq is required but not installed or not found in PATH."
        return 1  # Exit the function with an error status
    fi

    # Extract values using jq
    local engine=$(echo "$json_input" | $jq_path -r '.Engine')
    local storage_sources=$(echo "$json_input" | $jq_path -r '.StorageSources // empty')
    local publishers=$(echo "$json_input" | $jq_path -r '.Publishers // empty')
    local count=$(echo "$json_input" | $jq_path -r '.Count // 1')
    local ROOT=$(git rev-parse --show-toplevel)
    local job_path="$ROOT/test/jobs"

    # Define the job name
    local name="${engine} Job With StorageSource:${storage_sources} and Publisher:${publishers}"

    # Initialize jobspec JSON
    local jobspec_json=$($jq_path -n --arg name "$name" --arg count "$count" '{
        "Name": $name,
        "Type": "batch",
        "Namespace": "default",
        "Count": ($count | tonumber),
        "Tasks": [{
          "Name": "main"
          }]
    }')

    # Determine the engine index based on the presence of storage sources and publishers
    local engine_index
    if [[ -n "$storage_sources" && -n "$publishers" ]]; then
        engine_index=2  # Both are present
    elif [[ -n "$storage_sources" && -z "$publishers" ]]; then
        engine_index=0  # Only storage sources are present
    elif [[ -z "$storage_sources" && -n "$publishers" ]]; then
        engine_index=1  # Only publishers are present
    else
        engine_index=3  # Neither is present
    fi

    # Fetch the appropriate engine configuration from a JSON file
    local engine_json=$($jq_path -c ".[$engine_index].Engine" "$job_path/templates/engines.json")
    if [[ -z "$engine_json" ]]; then
        echo "Failed to fetch engine configuration."
        return 1
    fi

    # Incorporate the engine configuration into the jobspec
    jobspec_json=$(echo "$jobspec_json" | $jq_path --argjson engine "$engine_json" '.Tasks[0].Engine = $engine')

    # Handle input sources
    if [[ -n "$storage_sources" ]]; then
        local input_sources_json=$($jq_path -c ".[] | select(.InputSources[].Source.Type == \"$storage_sources\") | .InputSources" "$job_path/templates/input_sources.json")
        jobspec_json=$(echo "$jobspec_json" | $jq_path --argjson inputs "$input_sources_json" '.Tasks[0].InputSources = $inputs')
    fi

    # Handle publishers
    if [[ -n "$publishers" ]]; then
        local publisher_json=$($jq_path -c ".[] | select(.Publisher.Type == \"$publishers\") | .Publisher" "$job_path/templates/publishers.json")
        local result_paths_json=$($jq_path -c ".[] | select(.Publisher.Type == \"$publishers\") | .ResultPaths" "$job_path/templates/publishers.json")
        jobspec_json=$(echo "$jobspec_json" | $jq_path --argjson pub "$publisher_json" --argjson paths "$result_paths_json" '.Tasks[0].Publisher = $pub | .Tasks[0].ResultPaths = $paths')
    fi

    # Define filename and save jobspec to a file
    local filename="$job_path/specs/Engine${engine}_StorageSources${storage_sources}_Publishers${publishers}_Count${count}.json"
    echo "$jobspec_json" | $jq_path . > "$filename"
    echo "Generated jobspec file: $filename"
}

before_all() {
  ROOT=$(git rev-parse --show-toplevel)
  BACALHAU_BINARY=$(find "$ROOT/bin" -name 'bacalhau' -print -quit)
  BACALHAU="$BACALHAU_BINARY"
  export BACALHAU_UPDATE_SKIPCHECKS=true
  export LOG_LEVEL=WARN
  # export BACALHAU_NODE_SERVERAPI_HOST='localhost'
  export PATH="$(dirname "$BACALHAU_BINARY"):$PATH"
}


after_each() {
    
    if [[ "$environment" =~ "spawn" ]]; then
        teardown_nodes
        clean_repo
    else
        clean_repo_external
    fi
}