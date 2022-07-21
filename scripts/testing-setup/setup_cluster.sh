#!/bin/bash

terraform apply --auto-approve

echo -n "Sleeping for 60s to allow ssh to start..."
sleep 60
echo "Done."

runRemote() {
  local remote_addr args script

  remote_addr=$1; shift
  script=$1; shift

# generate eval-safe quoted version of current argument list
  printf -v args '%q ' "$@"

# pass that through on the command line to bash -s
# note that $args is parsed remotely by /bin/sh, not by bash!
  ssh -o LogLevel=ERROR -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" ubuntu@"$remote_addr" "sudo $script"
  # ssh -o LogLevel=ERROR -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" ubuntu@"$remote_addr" "sudo bash -s -x -- $args" < "$script"
}

all_nodes_public=()
while IFS='' read -r line; do all_nodes_public+=("$line"); done <  <(terraform output -json | jq -r '.instance_public_dns.value | .[] ')

all_nodes_private=()
while IFS='' read -r line; do all_nodes_private+=("$line"); done <  <(terraform output -json | jq -r '.instance_private_ips.value | .[] ')

first_node="${all_nodes_public[1]}"
echo "Connecting to: ubuntu@$first_node"
runRemote "$first_node" "/usr/local/bin/scripts/setup_node.sh"
runRemote "$first_node" "touch /tmp/remote_peer_string"

while true ; do
  peer_string=$(curl -s "$first_node/peer_token.html" | head -1)
  if [[ "$peer_string" == *"html"* ]]; then
    sleep 5
  else
    break
  fi
done


index=1
len_nodes=${#all_nodes_public[@]}

for i in "${!all_nodes_public[@]}"; do
    if (( index >= len_nodes)); then
      break
    fi

    this_node_public="${all_nodes_public[((i+1))]}"
    last_node_private="${all_nodes_private[((i))]}"

    echo "Peer string: $peer_string"

    echo "Connecting to: ubuntu@$this_node_public"
    runRemote "$this_node_public" "echo \"$peer_string\" > /tmp/remote_peer_string"
    runRemote "$this_node_public" "/usr/local/bin/scripts/setup_node.sh"
    ((index=index+1))
done
