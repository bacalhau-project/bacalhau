export IFS=$'\n\t'

new_repo() {
    export BACALHAU_DIR=$(mktemp -d)
    bacalhau id >/dev/null 2>&1
}

clean_repo() {
    rm -rf $BACALHAU_DIR
}

declare -i RUNNING_NODES=0

create_node() {
    TYPE=$1
    shift 1
    $BACALHAU serve --peer=none --node-type=$TYPE $@ 1>$BACALHAU_DIR/out.log 2>$BACALHAU_DIR/err.log &
    NODE_PID=$!
    RUNNING_NODES+=1
    {
        while ! ls $BACALHAU_DIR/bacalhau.run 2>/dev/null; do
            if ! ps $NODE_PID; then
                echo "$BACALHAU serve --peer=none --node-type=$TYPE $@ failed to start?" 1>&2;
                echo `$BACALHAU serve --peer=none --node-type=$TYPE $@` 1>&2
                exit 1
            fi
        done
    } 1>/dev/null

    # Ensure subsequent nodes automatically connect to this requester, and pick
    # a random port for the HTTP API to avoid collisions
    if test "$TYPE" = "requester"; then
        source $BACALHAU_DIR/bacalhau.run
        export BACALHAU_NODE_SERVERAPI_PORT=0
    fi
}

teardown_nodes() {
    for i in $(seq 1 +1 $RUNNING_NODES 2>/dev/null || true); do
        kill -15 %$i
        wait %$i 2>/dev/null
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
