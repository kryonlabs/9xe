#!/usr/bin/env bash
# Test script for working 9xe binaries
# This script tests all binaries that are known to work

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track results
PASSED=0
FAILED=0

# The 9xe binary is in the project root
cd "$PROJECT_DIR"

# Create test directory
echo "Setting up test environment..."
TMP_DIR="/tmp/9xe_test_$$"
mkdir -p "$TMP_DIR"
echo "test content" > "$TMP_DIR/test.txt"
echo "line 1" > "$TMP_DIR/file1.txt"
echo "line 2" > "$TMP_DIR/file2.txt"
echo "line 3" > "$TMP_DIR/file3.txt"

echo -e "\n${YELLOW}=== Testing Working Binaries ===${NC}"

# Test ls
echo -e "\n${YELLOW}Test 1: ls - list directory${NC}"
echo "Description: Should list files in directory"
if ./9xe /mnt/storage/Projects/TaijiOS/9xe/testbin/ls "$TMP_DIR" 2>&1 | strings | grep -E "file1.txt|file2.txt|file3.txt" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ PASSED${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAILED${NC}"
    FAILED=$((FAILED + 1))
fi

# Test cat
echo -e "\n${YELLOW}Test 2: cat - read file${NC}"
echo "Description: Should read and display file content"
if ./9xe /mnt/storage/Projects/TaijiOS/9xe/testbin/cat "$TMP_DIR/test.txt" 2>&1 | strings | grep "test content" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ PASSED${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAILED${NC}"
    FAILED=$((FAILED + 1))
fi

# Test date
echo -e "\n${YELLOW}Test 3: date - display time${NC}"
echo "Description: Should display current date/time"
if ./9xe /mnt/storage/Projects/TaijiOS/9xe/testbin/date 2>&1 | strings | grep -E "202[0-9]" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ PASSED${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAILED${NC}"
    FAILED=$((FAILED + 1))
fi

# Test ls with /tmp
echo -e "\n${YELLOW}Test 4: ls - list /tmp${NC}"
echo "Description: Should list /tmp directory"
if ./9xe /mnt/storage/Projects/TaijiOS/9xe/testbin/ls /tmp 2>&1 | strings | grep -E "test_ls|9xe_test" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ PASSED${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗ FAILED${NC}"
    FAILED=$((FAILED + 1))
fi

# Clean up
echo -e "\nCleaning up test files..."
rm -rf "$TMP_DIR"

# Print summary
TOTAL=$((PASSED + FAILED))
echo -e "\n${YELLOW}=== Test Summary ===${NC}"
echo -e "Total tests: $TOTAL"
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "\n${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}Some tests failed!${NC}"
    exit 1
fi
