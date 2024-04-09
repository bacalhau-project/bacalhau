#!bin/bashtub

source bin/bacalhau.sh

run_test() {
    WORD=$RANDOM
    subject bacalhau config set node.labels key=value "random=$WORD"
    create_node $1

    # Wait for node to have published information.
    subject bacalhau node list --output=json
    while ! jq -e 'length > 0' <<< $stdout 1>/dev/null; do
        sleep 0.05;
        subject bacalhau node list --output=json
    done

    assert_equal 1 $(jq -rcM length <<< $stdout)
    assert_not_equal 0 $(jq -rcM '.[0].Labels | length' <<< $stdout)
    assert_equal false $(jq -rcM '.[0].Labels["Operating-System"] == null' <<< $stdout)
    assert_equal false $(jq -rcM '.[0].Labels["Architecture"] == null' <<< $stdout)
    assert_equal value $(jq -rcM '.[0].Labels["key"]' <<< $stdout)
    assert_equal $WORD $(jq -rcM '.[0].Labels["random"]' <<< $stdout)
}

testcase_receive_labels_about_requester_node_for_nats() {
    subject bacalhau config set node.network.type nats
    assert_equal 0 $status
    run_test requester
}

testcase_receive_extra_labels_about_compute_node_for_nats() {
    subject bacalhau config set node.network.type nats
    assert_equal 0 $status
    run_test requester,compute
    assert_equal false $(jq -rcM '.[0].Labels["git-lfs"] == null' <<< $stdout)
}

testcase_receive_labels_about_requester_node_for_libp2p() {
    subject bacalhau config set node.network.type libp2p
    assert_equal 0 $status
    run_test requester
}

testcase_receive_extra_labels_about_compute_node_for_libp2p() {
    subject bacalhau config set node.network.type libp2p
    assert_equal 0 $status
    run_test requester,compute
    assert_equal false $(jq -rcM '.[0].Labels["git-lfs"] == null' <<< $stdout)
}
