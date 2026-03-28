#!/bin/bash
# Test runner wrapper for 9xe

XE_PATH="/home/wao/Projects/TaijiOS/9xe/9xe"
TEST_DIR="/home/wao/Projects/TaijiOS/9xe/test"

cd "$TEST_DIR" || exit 1

# Build the runner
echo "Building test runner..."
go build -o runner runner.go || {
    echo "Failed to build runner"
    exit 1
}

# Run tests
if [ $# -eq 0 ]; then
    # Run default tests
    echo "Running default test suite..."
    ./runner "$XE_PATH"
elif [ "$1" == "--all" ]; then
    # Run all basic utilities
    echo "Running all basic utilities..."
    ./runner "$XE_PATH" echo cat pwd date ls mkdir
else
    # Run specific binaries
    echo "Running tests for: $@"
    ./runner "$XE_PATH" "$@"
fi
