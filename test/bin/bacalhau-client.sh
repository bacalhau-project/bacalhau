
create_client() {
environment=$1

case "$environment" in
  production)
    echo "Running in production environment"
    export BACALHAU_DIR=/home/frrist/workspace/src/github.com/bacalhau-project/bacalhau/test/clusters/production
    ;;
  staging)
    echo "Running in staging environment"
    export BACALHAU_DIR=/home/frrist/workspace/src/github.com/bacalhau-project/bacalhau/test/clusters/staging
    ;;
  development)
    echo "Running in development environment"
    export BACALHAU_DIR=/home/frrist/workspace/src/github.com/bacalhau-project/bacalhau/test/clusters/development
    ;;
  local)
    echo "Running in local environment"
    export BACALHAU_DIR=/home/frrist/workspace/src/github.com/bacalhau-project/bacalhau/test/clusters/local
    # uses an existing cluster, doesn't delete anything after test runs
    ;;
  spawn)
    echo "Spawning a new environment"
    # TODO create a cluster and run the tests against that.
    # spawn an emphermal cluster and tear it down when tests are complete
    ;;
  *)
    echo "Unknown environment: $environment"
    ;;
esac
}

before_all() {
    ROOT=$(git rev-parse --show-toplevel)
    BACALHAU_BINARY=$(find $ROOT/bin -name 'bacalhau')
    BACALHAU=$BACALHAU_BINARY
    export LOG_LEVEL=WARN
    # export BACALHAU_NODE_SERVERAPI_HOST='localhost'
    export PATH=$(dirname $BACALHAU_BINARY):$PATH
}

after_each() {
    # TODO this function should remove everything from the repo execept the config file and token
    clean_repo
}
