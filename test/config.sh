#!bin/bashtub

source bin/bacalhau.sh

testcase_config_file_remains_empty_after_list() {
    subject bacalhau config list
    assert_equal 0 $status

    subject ls $BACALHAU_DIR/config.yaml
    assert_not_equal 0 $status
}

testcase_config_set_is_persistent() {
    TEST_VALUE=$RANDOM
    subject bacalhau config set 'User.InstallationID' $TEST_VALUE
    assert_equal 0 $status

    subject file $BACALHAU_DIR/config.yaml
    assert_equal 0 $status

    subject bacalhau config list --output=csv
    assert_match "user.installationid,$TEST_VALUE" "$stdout"
}