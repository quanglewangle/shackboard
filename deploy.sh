#!/bin/bash
# Usage: ./deploy.sh user@yourserver.example.com
set -e

SERVER=${1:-peter@fimblefowl.co.uk}

HASH=$(git rev-parse --short HEAD)

echo "Building for linux/amd64 (static, hash=$HASH)..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-X main.buildHash=$HASH" \
  -o /tmp/shackboard_deploy .

echo "Copying binary to $SERVER..."
scp /tmp/shackboard_deploy "$SERVER":/tmp/shackboard_new

echo "Installing on $SERVER..."
ssh "$SERVER" "mv /tmp/shackboard_new /home/peter/shackboard && systemctl --user restart shackboard && systemctl --user is-active shackboard"

echo "Done — deployed $HASH to $SERVER"
