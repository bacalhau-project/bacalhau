#!/bin/sh

# Check if nix-daemon is already running
if ! pgrep -x "nix-daemon" > /dev/null; then
    echo "Starting nix-daemon..."
    /nix/var/nix/profiles/default/bin/nix-daemon --daemon &
else
    echo "nix-daemon is already running."
fi

# Source Nix profile
. /nix/var/nix/profiles/default/etc/profile.d/nix.sh

# Execute the provided command
exec "$@"
