#!/usr/bin/env bash
# Simple build script for 9xe

echo "Building 9xe..."
nix-shell --run "go build -buildvcs=false -o 9xe ./cmd/9xe"

if [ $? -eq 0 ]; then
    echo "✓ Build successful: ./9xe"
    ls -lh 9xe
else
    echo "✗ Build failed"
    exit 1
fi
