#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

DIR="$(dirname "$0")"
cd $DIR/../../

SESSION=bacalhau-dashboard
export APP=${APP:=""}
export PREDICTABLE_API_PORT=1
export BACALHAU_API_HOST=localhost
export BACALHAU_API_PORT=20000
export BACALHAU_DASHBOARD_POSTGRES_HOST=127.0.0.1
export BACALHAU_DASHBOARD_POSTGRES_DATABASE=postgres
export BACALHAU_DASHBOARD_POSTGRES_USER=postgres
export BACALHAU_DASHBOARD_POSTGRES_PASSWORD=postgres
export BACALHAU_DASHBOARD_JWT_SECRET=apples
export DEVSTACK_ENV_FILE=$(mktemp)

function start() {
  if tmux has-session -t "$SESSION" 2>/dev/null; then
    echo "Session $SESSION already exists. Attaching..."
    sleep 1
    tmux -2 attach -t $SESSION
    exit 0;
  fi

  echo "Finding your Client ID..."
  export BACALHAU_JOB_APPROVER=$(bacalhau id | jq -r '.ClientID')
  export LOG_LEVEL=DEBUG

  rm -f /tmp/bacalhau-devstack.{port,pid}

  # get the size of the window and create a session at that size
  echo "Creating tmux session $SESSION..."
  local screensize=$(stty size)
  local width=$(echo -n "$screensize" | awk '{print $2}')
  local height=$(echo -n "$screensize" | awk '{print $1}')
  tmux -2 new-session -d -s $SESSION -x "$width" -y "$(($height - 1))"

  # the right hand col with a 50% vertical split
  tmux split-window -h -d
  tmux select-pane -t 1
  tmux split-window -v -d
  tmux select-pane -t 0
  tmux split-window -v -d
  tmux split-window -v -d
  tmux select-pane -t 4
  tmux split-window -v -d

  tmux send-keys -t 0 'go run . devstack --job-selection-probe-http=http://localhost:8081/api/v1/jobs/shouldrun'
  tmux send-keys -t 0 ' --hybrid-nodes=1 --requester-nodes=0 --compute-nodes=0 --job-selection-accept-networked' C-m
  tmux send-keys -t 1 'cd dashboard/frontend' C-m
  tmux send-keys -t 1 'yarn dev' C-m
  tmux send-keys -t 2 'cd dashboard/api' C-m
  tmux send-keys -t 2 'while ! (test -f "$DEVSTACK_ENV_FILE" && grep BACALHAU_PEER_CONNECT $DEVSTACK_ENV_FILE); do sleep 1; done; '
  tmux send-keys -t 2 'source $DEVSTACK_ENV_FILE; go run . serve --port 8081 --peer $BACALHAU_PEER_CONNECT' C-m
  tmux send-keys -t 3 "bacalhau docker run --network=HTTP --domain=.gov.uk curl curlimages/curl https://data.gov.uk"
  tmux send-keys -t 4 "cd dashboard/api" C-m
  tmux send-keys -t 4 "go run . user add --username=test --password=password"
  tmux send-keys -t 5 "docker run -ti --rm --name postgres -p 5432:5432 -e POSTGRES_DB=$BACALHAU_DASHBOARD_POSTGRES_DATABASE -e POSTGRES_USER=$BACALHAU_DASHBOARD_POSTGRES_USER -e POSTGRES_PASSWORD=$BACALHAU_DASHBOARD_POSTGRES_PASSWORD postgres" C-m

  tmux -2 attach-session -t $SESSION
}

function stop() {
  echo "Stopping tmux session $SESSION..."
  docker rm -f postgres || true
  tmux kill-session -t $SESSION
}

command="$@"

if [ -z "$command" ]; then
  command="start"
fi

eval "$command"
