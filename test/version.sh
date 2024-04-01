#!bin/bashtub

source bin/bacalhau-client.sh

testcase_can_list_version() {
    create_client "$CLUSTER"
    # run the version command to initialize the repo and supress any errors around ClientID
    subject bacalhau version
    assert_equal 0 "${status}"
    VERSION_INFO=$(curl -s https://get.bacalhau.org/version)
    EXPECTED_GITVERSION=$(echo "$VERSION_INFO" | jq -r '.version.gitversion')
    subject bacalhau version --output json
    if [[ "$CLUSTER" == "spawn" ]]; then
        ACTUAL_GITVERSION=$(echo $stdout | jq -r '.latestVersion.GitVersion')
    else
        ACTUAL_GITVERSION=$(echo $stdout | jq -r '.serverVersion.GitVersion')
    fi
    assert_equal 0 $status
    assert_equal "$EXPECTED_GITVERSION" "$ACTUAL_GITVERSION"
    assert_equal '' $stderr
}