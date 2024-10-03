#!bin/bashtub

source bin/bacalhau.sh

# assert we can start bacalhau with two config files, the latter overriding and merging with the base
testcase_config_with_override_file() {
  start_bacalhau_serve_with_config -c $ROOT/testdata/config/base.yaml -c $ROOT/testdata/config/override.yaml

  subject curl -s http://localhost:1234/api/v1/agent/config
  assert_equal 0 $status > /dev/null

  # These are default values unaffected by the config file
  assert_match "0.0.0.0" $(echo $stdout | jq .API.Host)
  assert_match 1234 $(echo $stdout | jq .API.Port)
  assert_match "true" $(echo $stdout | jq .Orchestrator.Enabled)
  assert_match "false" $(echo $stdout | jq .Compute.Enabled)

  # These are values set in the base config file that were not overridden
  assert_match "bar" $(echo $stdout | jq .Compute.Labels.foo)


  # This value overrides the base config
  assert_match "hostname" $(echo $stdout | jq .NameProvider)

  # This value is merged between the configs
  assert_match "boo" $(echo $stdout | jq .Compute.Labels.buz)

  kill $SERVER_PID
}
