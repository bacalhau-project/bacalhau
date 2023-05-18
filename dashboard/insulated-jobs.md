# Demo

1. `bash scripts/tmux.sh`
2. Restart the devstack with:
    - DEVSTACK_ENV_FILE set to some temp location
    - `export BACALHAU_JOB_APPROVER=$(go run . id | jq -r '.ClientID')` to set to your Client ID
    - `--job-selection-probe-http=http://localhost:8081/api/v1/jobs/shouldrun`
    - some permissive job selection policy e.g. `--job-selection-accept-networked`
3. Run the API with:
    - source $DEVSTACK_ENV_FILE;
    - go run . serve --port 8081 --peer $BACALHAU_PEER_CONNECT
4. Set up a user with:
    - cd dashboard/pkg/api;
    - go run . user add --username=simon password=password
5. Run a moderated jobs with:
    - DEVSTACK_ENV_FILE equal to the temp location
    - go run . docker run --network=HTTP --domain=.gov.uk curl curlimages/curl https://data.gov.uk
