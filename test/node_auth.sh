#!bin/bashtub

source bin/bacalhau.sh

testcase_node_can_connect_without_token() {
    create_node orchestrator

    create_node compute
    # If this returns successfully, the node started and authenticated.
}

testcase_preconfigured_token_not_printed() {
    subject ${BACALHAU} config set orchestrator.auth.token kerfuffle
    assert_equal 0 $status

    create_node orchestrator
    assert_equal 0 $status
    assert_not_match kerfuffle $stdout
}

testcase_node_connects_with_preconfigured_token() {
    subject ${BACALHAU} config set orchestrator.auth.token kerfuffle
    assert_match 0 $status
    create_node orchestrator

    subject ${BACALHAU} config set orchestrator.auth.token kerfuffle
    create_node compute
    # If this returns successfully, the node started and authenticated.
}

testcase_node_connects_with_url_embedded_token() {
    subject ${BACALHAU} config set orchestrator.auth.token kerfuffle
    assert_match 0 $status
    create_node orchestrator


    # remove the token from the config
    subject ${BACALHAU} config set orchestrator.auth.token ""

    # add the token to the orchestrator URL
    export BACALHAU_NODE_NETWORK_ORCHESTRATORS=$(echo $BACALHAU_NODE_NETWORK_ORCHESTRATORS | sed "s|nats://|nats://kerfuffle@|")

    create_node compute
    # If this returns successfully, the node started and authenticated.
}

testcase_node_cannot_connect_with_wrong_token() {
    subject ${BACALHAU} config set orchestrator.auth.token kerfuffle
    assert_match 0 $status
    create_node orchestrator

    export BACALHAU_DIR=$(mktemp -d)
    subject ${BACALHAU} serve --compute 1>$BACALHAU_DIR/out.log 2>$BACALHAU_DIR/err.log
    assert_not_equal 0 $status
    assert_match "nats: Authorization Violation" $stdout
}
