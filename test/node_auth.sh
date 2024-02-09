#!bin/bashtub

source bin/bacalhau.sh

testcase_node_can_connect_with_correct_token() {
    subject bacalhau config set node.network.type nats
    create_node requester

    # Check that the orchestrator URL contains an '@' which shows that it has
    # auth credentials included
    subject grep BACALHAU_NODE_NETWORK_ORCHESTRATORS $BACALHAU_DIR/bacalhau.run
    assert_match '@' $stdout

    new_repo
    create_node compute
    # If this returns successfully, the node started and authenticated.
}

testcase_node_connects_with_preconfigured_token() {
    subject bacalhau config set node.network.authsecret kerfuffle
    assert_match 0 $status
    create_node requester

    # Remove auth token from orchestrator URL
    export BACALHAU_NODE_NETWORK_ORCHESTRATORS=$(echo $BACALHAU_NODE_NETWORK_ORCHESTRATORS | sed "s:[^\/]*@::")
    new_repo
    subject bacalhau config set node.network.authsecret kerfuffle
    create_node compute
    # If this returns successfully, the node started and authenticated.
}

testcase_node_cannot_connect_without_token() {
    subject bacalhau config set node.network.type nats
    create_node requester

    subject grep BACALHAU_NODE_NETWORK_ORCHESTRATORS $BACALHAU_DIR/bacalhau.run
    assert_match '@' $stdout

    # Remove auth token from orchestrator URL
    export BACALHAU_NODE_NETWORK_ORCHESTRATORS=$(echo $BACALHAU_NODE_NETWORK_ORCHESTRATORS | sed "s:[^\/]*@::")
    new_repo
    subject bacalhau serve --node-type compute
    assert_not_equal 0 $status
    assert_match "nats: Authorization Violation" $stderr
}
