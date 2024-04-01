#!bin/bashtub

source bin/bacalhau-client.sh

testcase_can_list_version() {
    create_client "$CLUSTER"
    VERSION_INFO=$(curl -s https://get.bacalhau.org/version)
    EXPECTED_GITVERSION=$(echo "$VERSION_INFO" | jq -r '.version.gitversion')
    subject bacalhau version --output json
    ACTUAL_GITVERSION=$(echo $stdout | jq -r '.latestVersion.GitVersion')
    assert_equal 0 $status
    assert_equal "$EXPECTED_GITVERSION" "$ACTUAL_GITVERSION"
    assert_equal '' $stderr
}