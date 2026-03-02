#!/usr/bin/env bash
set -euo pipefail
mkdir -p dist

echo "Building Linux binary..."
GOOS=linux GOARCH=amd64 go build -o dist/netscope-linux-amd64 ./cmd/netscope

echo "Building Windows binary..."
GOOS=windows GOARCH=amd64 go build -o dist/netscope-windows-amd64.exe ./cmd/netscope

echo "Done. Binaries in dist/"
