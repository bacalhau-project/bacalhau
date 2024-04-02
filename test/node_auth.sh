#!bin/bashtub

source bin/bacalhau.sh

testcase_node_can_connect_without_token() {
    subject bacalhau config set node.network.type nats
    create_node requester

    create_node compute
    # If this returns successfully, the node started and authenticated.
}

testcase_preconfigured_token_not_printed() {
    subject bacalhau config set node.network.type nats
    subject bacalhau config set node.network.authsecret kerfuffle
    assert_equal 0 $status

    create_node requester
    assert_equal 0 $status

    # check stdout
    subject grep BACALHAU_NODE_NETWORK_ORCHESTRATORS $BACALHAU_DIR/out.log
    assert_equal 0 $status
    assert_not_match kerfuffle $stdout

    # check bacalhau.run
    subject grep BACALHAU_NODE_NETWORK_ORCHESTRATORS $BACALHAU_DIR/bacalhau.run
    assert_equal 0 $status
    assert_not_match kerfuffle $stdout
}

testcase_node_connects_with_preconfigured_token() {
    subject bacalhau config set node.network.type nats
    subject bacalhau config set node.network.authsecret kerfuffle
    assert_match 0 $status
    create_node requester

    subject bacalhau config set node.network.authsecret kerfuffle
    subject bacalhau config set node.network.type nats
    create_node compute
    # If this returns successfully, the node started and authenticated.
}

testcase_node_connects_with_url_embedded_token() {
    subject bacalhau config set node.network.type nats
    subject bacalhau config set node.network.authsecret kerfuffle
    assert_match 0 $status
    create_node requester


    # remove the token from the config
    subject bacalhau config set node.network.authsecret ""
    subject bacalhau config set node.network.type nats

    # add the token to the orchestrator URL
    export BACALHAU_NODE_NETWORK_ORCHESTRATORS=$(echo $BACALHAU_NODE_NETWORK_ORCHESTRATORS | sed "s|nats://|nats://kerfuffle@|")

    create_node compute
    # If this returns successfully, the node started and authenticated.
}

testcase_node_cannot_connect_with_wrong_token() {
    subject bacalhau config set node.network.type nats
    subject bacalhau config set node.network.authsecret kerfuffle
    assert_match 0 $status
    create_node requester

    subject bacalhau config set node.network.type nats
    subject bacalhau config set node.network.authsecret kerfalafel
    subject bacalhau serve --node-type compute
    assert_not_equal 0 $status
    assert_match "nats: Authorization Violation" $stderr
}
