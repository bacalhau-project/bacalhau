#!bin/bashtub

source bin/bacalhau.sh

testcase_config_with_file() {
  start_bacalhau_serve_with_config -c $ROOT/testdata/config/base.yaml

  subject curl -s http://localhost:1234/api/v1/agent/config
  assert_equal 0 $status

  # These are default values unaffected by the config file
  assert_match "0.0.0.0" $(echo $stdout | jq .API.Host)
  assert_match 1234 $(echo $stdout | jq .API.Port)
  assert_match "true" $(echo $stdout | jq .Orchestrator.Enabled)
  assert_match "false" $(echo $stdout | jq .Compute.Enabled)

  # These are values set in the config file
  assert_match "uuid" $(echo $stdout | jq .NameProvider)
  assert_match "bar" $(echo $stdout | jq .Labels.foo)

  kill $SERVER_PID
}
