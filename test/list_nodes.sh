#!bin/bashtub

source bin/bacalhau-client.sh

testcase_can_list_nodes_and_count() {
    
    create_client "$CLUSTER"
    echo "Checking $BACALHAU_DIR/metadata.json"
    if [[ -s "$BACALHAU_DIR/metadata.json" ]]; then
        node_count=$(jq 'length' "$BACALHAU_DIR/metadata.json")
        echo "Node count: $node_count"
    else
        echo "Error: $BACALHAU_DIR/metadata.json does not exist or is empty."
        # Debug: Show if the file exists and its size
        ls -l "$BACALHAU_DIR/metadata.json"
    fi

    subject bacalhau node list --output json | awk '/^\[|^{/ {print; inJson=1; next} inJson {print}'
    assert_equal 0 $status
    assert_match $node_count $(echo $stdout | jq length)
    assert_equal '' $stderr
}