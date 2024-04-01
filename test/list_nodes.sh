#!bin/bashtub

source bin/bacalhau-client.sh

testcase_can_list_nodes_and_count() {
    
    create_client "$CLUSTER"
    # run the version command to initialize the repo and supress any errors around ClientID
    subject bacalhau version
    assert_equal 0 "${status}"
    echo "Checking $BACALHAU_DIR/metadata.json"
    if [[ "$CLUSTER" == "spawn" ]]; then
        node_count=1
        echo "Cluster is 'spawn', setting node count to $node_count."
    else
        if [[ -s "$BACALHAU_DIR/metadata.json" ]]; then
            node_count=$(jq 'length' "$BACALHAU_DIR/metadata.json")
            echo "Node count: $node_count"
        else
            echo "Error: $BACALHAU_DIR/metadata.json does not exist or is empty."
            # Debug: Show if the file exists and its size
            ls -l "$BACALHAU_DIR/metadata.json"
        fi
    fi

    subject bacalhau node list --output json
    assert_equal 0 $status
    assert_match $node_count $(echo $stdout | jq length)
    assert_equal '' $stderr
}