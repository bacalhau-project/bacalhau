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
  local InputSources=$(echo "$json_input" | jq -r '.InputSources | ascii_downcase')
  local Publisher=$(echo "$json_input" | jq -r '.Publisher | ascii_downcase')
  local count=$(echo "$json_input" | jq -r '.count')

  # echo "Testing with Engine=$Engine, InputSources=$InputSources, Publisher=$Publisher, count=$count"

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
    local publisher_match=$(echo "$metadata" | jq -r ".[$i].ComputeNodeInfo.Publishers[] | ascii_downcase | select(. == \"$Publisher\")")
    local source_match=$(echo "$metadata" | jq -r ".[$i].ComputeNodeInfo.StorageSources[] | ascii_downcase | select(. == \"$InputSources\")")

    # echo "Node $i: Engine match=$engine_match, Publisher match=$publisher_match, Input Source match=$source_match"

    if [[ -n "$engine_match" ]] && 
       ([[ -z "$Publisher" ]] || [[ -n "$publisher_match" ]]) && 
       ([[ -z "$InputSources" ]] || [[ -n "$source_match" ]]); then
      ((match_count++))
    fi
  done

  # echo "Total matches found: $match_count"
  if [[ "$match_count" -ge "$count" ]]; then
    echo "true"
    return 0
  else
    echo "false"
    return 1
  fi
}

generate_jobspec() {
  jq_installed=$(command -v jq)
  if [ -z "$jq_installed" ]; then
    echo "jq is required but it's not installed. Please install jq to use this script."
    exit 1
  fi
  # The first argument is a JSON string
  local json_input="$1"
  echo "$json_input"
  # Extract values from JSON string
  local engine=$(echo "$json_input" | jq -r ".Engine")
  local storage_sources=$(echo "$json_input" | jq -r ".StorageSources")
  local publishers=$(echo "$json_input" | jq -r ".Publishers")
  local count=$(echo "$json_input" | jq -r ".Count")
  local ROOT=$(git rev-parse --show-toplevel)
  local PATH=$ROOT/test/jobs
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
  local engine_json=$(jq -c ".[$engine_index].Engine" $PATH/templates/engines.json)

  # Add engine JSON to the jobspec and include task name "main"
  jobspec_json=$(echo "$jobspec_json" | jq --argjson engine "$engine_json" \
    '.Tasks[0] = {"Name": "main", "Engine": $engine}')

  # Handle input sources
  if [[ -n "$storage_sources" ]]; then
    local input_sources_json=$(jq -c ".[] | select(.InputSources[].Source.Type == \"$storage_sources\") | .InputSources" $PATH/templates/input_sources.json)
    jobspec_json=$(echo "$jobspec_json" | jq --argjson inputs "$input_sources_json" '.Tasks[0].InputSources = $inputs')
  fi

  # Handle publishers
  if [[ -n "$publishers" ]]; then
    local publisher_json=$(jq -c ".[] | select(.Publisher.Type == \"$publishers\")" $PATH/templates/publishers.json)
    jobspec_json=$(echo "$jobspec_json" | jq --argjson pub "$publisher_json.Publisher" '.Tasks[0].Publisher = $pub')
    jobspec_json=$(echo "$jobspec_json" | jq --argjson paths "$publisher_json.ResultPaths" '.Tasks[0].ResultPaths = $paths')
  fi

  # Define filename based on parameters
  local filename="Engine${engine}_StorageSources${storage_sources}_Publishers${publishers}_Count${count}.json"

  # Output the complete jobspec JSON to a file
  echo "$jobspec_json" | jq . > "$PATH/specs/$filename"
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