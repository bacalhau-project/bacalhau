source bin/bacalhau.sh

clean_repo_external() {
find "${BACALHAU_DIR}" -mindepth 1 \( ! -name 'config.yaml' ! -name 'token.txt' \) -exec rm -rf {} +
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
  *)
    echo "Unknown environment: $environment"
    ;;
esac
}

before_all() {
  ROOT=$(git rev-parse --show-toplevel)
  BACALHAU_BINARY=$(find "$ROOT/bin" -name 'bacalhau' -print -quit)
  BACALHAU="$BACALHAU_BINARY"
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