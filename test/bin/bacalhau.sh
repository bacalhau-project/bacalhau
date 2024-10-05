export IFS=$'\n\t'

declare -a RUNNING_NODES

new_repo() {
    export BACALHAU_DIR=$(mktemp -d)
    export BACALHAU_UPDATE_SKIPCHECKS=true
    export BACALHAU_NODE_COMPUTE_LOCALPUBLISHER_ADDRESS=127.0.0.1
    RUNNING_NODES=()
    bacalhau id >/dev/null 2>&1
}

clean_repo() {
    rm -rf $BACALHAU_DIR
}

create_node() {
    TYPE=$1
    shift 1

    if [[ "$TYPE" == *","* ]]; then
        # Split the types and run the command with both
        IFS=',' read -ra TYPES <<< "$TYPE"
        $BACALHAU serve --"${TYPES[0]}" --"${TYPES[1]}" "$@" 1>$BACALHAU_DIR/out.log 2>$BACALHAU_DIR/err.log &
        echo $BACALHAU serve --"${TYPES[0]}" --"${TYPES[1]}" "$@" 1>$BACALHAU_DIR/out.log 2>$BACALHAU_DIR/err.log
    else
        # Run the command with the single type
        $BACALHAU serve --"$TYPE" "$@" 1>$BACALHAU_DIR/out.log 2>$BACALHAU_DIR/err.log &
        echo $BACALHAU serve --"$TYPE" "$@" 1>$BACALHAU_DIR/out.log 2>$BACALHAU_DIR/err.log
    fi

    NODE_PID=$!
    RUNNING_NODES+=($NODE_PID)
    {
        while ! ls $BACALHAU_DIR/bacalhau.run 2>/dev/null; do
            if ! ps $NODE_PID; then
                echo "$BACALHAU serve --node-type=$TYPE $@ failed to start?" 1>&2;
                echo `$BACALHAU serve --node-type=$TYPE $@` 1>&2
                exit 1
            fi
            sleep 0.01
        done
    } 1>/dev/null

    # Ensure subsequent nodes automatically connect to this requester, and pick
    # a random port for the HTTP API to avoid collisions
    if [[ "$TYPE" =~ "requester" ]]; then
        source $BACALHAU_DIR/bacalhau.run
        export BACALHAU_NODE_SERVERAPI_PORT=0
    fi
}

teardown_nodes() {
    for i in $RUNNING_NODES; do
        while kill -15 $i 1>/dev/null 2>&1; do
            sleep 0.01
        done
    done;
}

require_docker() {
    subject docker info
    assert_equal 0 $status
}

before_all() {
    ROOT=$(git rev-parse --show-toplevel)
    BACALHAU_BINARY=$(find $ROOT/bin -name 'bacalhau')
    BACALHAU=$BACALHAU_BINARY
    export LOG_LEVEL=WARN
    # export BACALHAU_NODE_SERVERAPI_HOST='localhost'
    export PATH=$(dirname $BACALHAU_BINARY):$PATH
}

before_each() {
    new_repo
}

after_each() {
    teardown_nodes
    clean_repo
}

start_bacalhau_serve_with_config() {
  # Start the server in the background
  $BACALHAU serve $@ > /dev/null 2>&1 &

  SERVER_PID=$!

  # Wait for the server to come online
  # You can use a loop to check if the server is responding, e.g., using curl
  while ! curl -s http://localhost:1234/api/v1/agent/alive; do
    echo "Waiting for bacalhau server to come online..."
    sleep 1
  done
}
