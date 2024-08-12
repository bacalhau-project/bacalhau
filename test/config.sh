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

# assert we can start bacalhau without a config and the default values are used
testcase_config_with_no_flags_has_defaults() {
  start_bacalhau_serve_with_config

  subject curl -s http://localhost:1234/api/v1/agent/config
  assert_equal 0 $status
  assert_match "0.0.0.0" $(echo $stdout | jq .Node.ServerAPI.Host)
  assert_match 1234 $(echo $stdout | jq .Node.ServerAPI.Port)
  assert_match "requester" $(echo $stdout | jq .Node.Type[])
  assert_match "bootstrap.production.bacalhau.org" $(echo $stdout | jq .Node.ClientAPI.Host)
  assert_match "1234" $(echo $stdout | jq .Node.ClientAPI.Port)
  assert_match "{}" $(echo $stdout | jq .Node.Labels)

  kill $SERVER_PID
}

# assert we can start bacalhau with a config file via the -c flag
testcase_config_with_file() {
  start_bacalhau_serve_with_config -c $ROOT/testdata/config/base.yaml

  subject curl -s http://localhost:1234/api/v1/agent/config
  assert_equal 0 $status

  # These are default values unaffected by the config file
  assert_match "0.0.0.0" $(echo $stdout | jq .Node.ServerAPI.Host)
  assert_match 1234 $(echo $stdout | jq .Node.ServerAPI.Port)
  assert_match "requester" $(echo $stdout | jq .Node.Type[])

  # These are values set in the config file
  assert_match "1.1.1.1" $(echo $stdout | jq .Node.ClientAPI.Host)
  assert_match 1111 $(echo $stdout | jq .Node.ClientAPI.Port)
  assert_match "bar" $(echo $stdout | jq .Node.Labels.foo)

  kill $SERVER_PID
}

# assert we can start bacalhau with two config files, the latter overriding and merging with the base
testcase_config_with_override_config_file() {
  start_bacalhau_serve_with_config -c $ROOT/testdata/config/base.yaml -c $ROOT/testdata/config/override.yaml

  subject curl -s http://localhost:1234/api/v1/agent/config
  assert_equal 0 $status > /dev/null

  # These are default values unaffected by the config file
  assert_match "0.0.0.0" $(echo $stdout | jq .Node.ServerAPI.Host)
  assert_match 1234 $(echo $stdout | jq .Node.ServerAPI.Port)
  assert_match "requester" $(echo $stdout | jq .Node.Type[])

  # These are values set in the base config file that were not overridden
  assert_match 1111 $(echo $stdout | jq .Node.ClientAPI.Port)
  assert_match "bar" $(echo $stdout | jq .Node.Labels.foo)


  # This value overrides the base config
  assert_match "2.2.2.2" $(echo $stdout | jq .Node.ClientAPI.Host)

  # This value is merged between the configs
  assert_match "boo" $(echo $stdout | jq .Node.Labels.buz)

  kill $SERVER_PID
}

# same as test case above, but with addition of --api-host flag that overrides both configs files
testcase_config_with_override_config_file_with_api_config_flag() {
  start_bacalhau_serve_with_config -c $ROOT/testdata/config/base.yaml -c $ROOT/testdata/config/override.yaml --api-host=3.3.3.3

  subject curl -s http://localhost:1234/api/v1/agent/config
  assert_equal 0 $status

  # These are default values unaffected by the config file
  assert_match "0.0.0.0" $(echo $stdout | jq .Node.ServerAPI.Host)
  assert_match 1234 $(echo $stdout | jq .Node.ServerAPI.Port)
  assert_match "requester" $(echo $stdout | jq .Node.Type[])

  # These are values set in the base config file that were not overridden
  assert_match 1111 $(echo $stdout | jq .Node.ClientAPI.Port)
  assert_match "bar" $(echo $stdout | jq .Node.Labels.foo)


  # This value overrides the config files from the flag
  assert_match "3.3.3.3" $(echo $stdout | jq .Node.ClientAPI.Host)

  # This value is merged between the configs
  assert_match "boo" $(echo $stdout | jq .Node.Labels.buz)

  kill $SERVER_PID
}

# same as test case above, but with addition of a config flag and the dedicated --api-flag, config flag takes precedence
testcase_config_with_override_config_file_with_api_config_flag_and_dedicated_flag() {
  start_bacalhau_serve_with_config -c $ROOT/testdata/config/base.yaml -c $ROOT/testdata/config/override.yaml --api-host=3.3.3.3 -c node.clientapi.host=4.4.4.4

  subject curl -s http://localhost:1234/api/v1/agent/config
  assert_equal 0 $status

  # These are default values unaffected by the config file
  assert_match "0.0.0.0" $(echo $stdout | jq .Node.ServerAPI.Host)
  assert_match 1234 $(echo $stdout | jq .Node.ServerAPI.Port)
  assert_match "requester" $(echo $stdout | jq .Node.Type[])

  # These are values set in the base config file that were not overridden
  assert_match 1111 $(echo $stdout | jq .Node.ClientAPI.Port)
  assert_match "bar" $(echo $stdout | jq .Node.Labels.foo)


  # This value overrides the config files from flag
  assert_match "4.4.4.4" $(echo $stdout | jq .Node.ClientAPI.Host)

  # This value is merged between the configs
  assert_match "boo" $(echo $stdout | jq .Node.Labels.buz)

  kill $SERVER_PID
}

start_bacalhau_serve_with_config() {
  # Start the server in the background
  $BACALHAU serve $@ > /dev/null 2>&1 &

  SERVER_PID=$!

  # Wait for the server to come online
  # You can use a loop to check if the server is responding, e.g., using curl
  while ! curl -s http://localhost:1234/api/v1/agent/alive; do
    echo "Waiting for bacalhau server to come online..."
    sleep 1
  done
}