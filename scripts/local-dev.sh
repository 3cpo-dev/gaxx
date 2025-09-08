#!/usr/bin/env bash
set -euo pipefail

echo "Building gaxx and gaxx-agent..."
CGO_ENABLED=0 go build -o bin/gaxx ./cmd/gaxx
CGO_ENABLED=0 go build -o bin/gaxx-agent ./cmd/gaxx-agent
echo "Done."


