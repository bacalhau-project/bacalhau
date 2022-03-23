#!/bin/bash
while true ; do
  peer_string=$(curl -s "ec2-54-194-9-231.eu-west-1.compute.amazonaws.com/peer_token.html" | head -1) 
  if [[ "$peer_string" == *"html"* ]]; then
    sleep 5
  else
    break
  fi
done

echo $peer_string