#!bin/bashtub

source bin/bacalhau.sh

run_test() {
    WORD=$RANDOM
    subject ${BACALHAU} config set labels "key=value,random=$WORD"
    create_node $1

    # Wait for node to have published information.
    subject ${BACALHAU} node list --output=json
    while ! jq -e 'length > 0' <<< $stdout 1>/dev/null; do
        sleep 0.05;
        subject ${BACALHAU} node list --output=json
    done

    assert_equal 1 $(jq -rcM length <<< $stdout)
    assert_not_equal 0 $(jq -rcM '.[0].Info.Labels | length' <<< $stdout)
    assert_equal false $(jq -rcM '.[0].Info.Labels["Operating-System"] == null' <<< $stdout)
    assert_equal false $(jq -rcM '.[0].Info.Labels["Architecture"] == null' <<< $stdout)
    assert_equal value $(jq -rcM '.[0].Info.Labels["key"]' <<< $stdout)
    assert_equal $WORD $(jq -rcM '.[0].Info.Labels["random"]' <<< $stdout)
}

testcase_receive_labels_about_node() {
    run_test compute,orchestrator
}
