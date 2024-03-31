#!bin/bashtub

source bin/bacalhau.sh

testcase_can_list_nodes_and_count() {
    create_node requester,compute

    subject bacalhau node list --output json
    assert_equal 0 $status
    assert_match '1' $(echo $stdout | jq length)
    assert_equal '' $stderr
}