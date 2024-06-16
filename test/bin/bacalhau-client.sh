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

  if [[ "$match_count" -ge "$count" ]]; then
    return 0
  else
    echo "Insufficient Resources: Required $count, but found $match_count"
    return 1
  fi
}

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