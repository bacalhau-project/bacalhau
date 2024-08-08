#!bin/bashtub

source bin/bacalhau.sh

testcase_config_set_is_persistent() {
    TEST_VALUE=$RANDOM
    subject bacalhau config set 'User.InstallationID' $TEST_VALUE
    assert_equal 0 $status

    subject file $BACALHAU_DIR/config.yaml
    assert_equal 0 $status

    # Verify the contents of the config file
    subject cat "$BACALHAU_DIR/config.yaml | grep installationid"
    assert_match "${TEST_VALUE}" "$stdout"
}