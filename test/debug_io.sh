#!/usr/bin/env bash
# Test script to understand I/O issues

echo "=== Testing 9xe with minimal I/O ==="

# Test date (known to work)
echo "1. Testing date (should work):"
timeout 2 ./9xe /mnt/storage/Projects/TaijiOS/9xe/testbin/date 2>&1 | tail -1

echo ""
echo "2. Testing echo with simple input:"
echo "Testing echo" | timeout 2 ./9xe /mnt/storage/Projects/TaijiOS/9xe/testbin/cat 2>&1 | grep -E "(echo|Testing|package main)" | head -5

echo ""
echo "3. Testing cat with runner.go:"
timeout 3 ./9xe /mnt/storage/Projects/TaijiOS/9xe/testbin/cat /home/wao/Projects/TaijiOS/9xe/test/runner.go 2>&1 | grep -E "^\[SYSCALL\]|package main|^\[sys\]" | head -10

echo ""
echo "4. Looking for infinite loops:"
timeout 2 ./9xe /mnt/storage/Projects/TaijiOS/9xe/testbin/cat /home/wao/Projects/TaijiOS/9xe/test/runner.go 2>&1 | grep -E "LOOP|infinite" | head -5

echo ""
echo "=== Test complete ==="
