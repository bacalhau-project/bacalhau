# #!/bin/bash

# port=54545

# until [[ -f /tmp/bacalhau_peer_token ]]; do
#   sleep 5
#   echo "No bacalhau_peer_token found..."
# done

# until [[ -f /tmp/bacalhau_peer_ip ]]; do
#   sleep 5
#   echo "No bacalhau_peer_ip found..."
# done

# serve_token=$(cat /tmp/bacalhau_peer_token)
# serve_ip=$(cat /tmp/bacalhau_peer_ip)

# if [[ -z "$serve_token" ]]
# then
#       bacalhau --jsonrpc-port $(port) serve
# else
#       bacalhau serve --peer /ip4/$(serve_ip)/tcp/0/p2p/$(serve_token) --jsonrpc-port $(port)
# fi