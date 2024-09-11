#!bin/bashtub

source bin/bacalhau.sh

testcase_config_with_override_config_file_and_flag() {
  start_bacalhau_serve_with_config -c $ROOT/testdata/config/base.yaml -c $ROOT/testdata/config/override.yaml --labels=apple=banana

  subject curl -s http://localhost:1234/api/v1/agent/config
  assert_equal 0 $status > /dev/null

  # These are default values unaffected by the config file
  assert_match "0.0.0.0" $(echo $stdout | jq .API.Host)
  assert_match 1234 $(echo $stdout | jq .API.Port)
  assert_match "true" $(echo $stdout | jq .Orchestrator.Enabled)
  assert_match "false" $(echo $stdout | jq .Compute.Enabled)

  # set in the override
  assert_match "hostname" $(echo $stdout | jq .NameProvider)

  # overritten by the flag
  assert_match "null" $(echo $stdout | jq .Compute.Labels.foo)
  assert_match "null" $(echo $stdout | jq .Compute.Labels.buz)


  # This value is merged between the configs
  assert_match "banana" $(echo $stdout | jq .Compute.Labels.apple)

  kill $SERVER_PID
}
