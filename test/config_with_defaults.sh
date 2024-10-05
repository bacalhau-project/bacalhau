#!bin/bashtub

source bin/bacalhau.sh

# assert we can start bacalhau without a config and the default values are used
testcase_config_with__defaults() {
  start_bacalhau_serve_with_config

  subject curl -s http://localhost:1234/api/v1/agent/config
  assert_equal 0 $status

  assert_match "puuid" $(echo $stdout | jq .NameProvider)
  assert_match "0.0.0.0" $(echo $stdout | jq .API.Host)
  assert_match 1234 $(echo $stdout | jq .API.Port)
  assert_match "true" $(echo $stdout | jq .Orchestrator.Enabled)
  assert_match "false" $(echo $stdout | jq .Compute.Enabled)
  assert_match "null" $(echo $stdout | jq .Compute.Labels)

  kill $SERVER_PID
}
