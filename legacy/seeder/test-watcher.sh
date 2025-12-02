#!/bin/bash
# File Watcher Integration Test Script
# Tests automatic package detection and seeding

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}Cleaning up test environment...${NC}"
    
    # Stop seeder if running
    if [ -f seeder.pid ]; then
        PID=$(cat seeder.pid)
        if kill -0 "$PID" 2>/dev/null; then
            echo "Stopping seeder (PID: $PID)..."
            kill -TERM "$PID" 2>/dev/null || true
            sleep 2
            kill -9 "$PID" 2>/dev/null || true
        fi
        rm -f seeder.pid
    fi
    
    # Clean test directories
    rm -rf ./packages/seeded ./packages/invalid
    rm -f ./packages/*.tar.gz ./packages/*.tgz
    rm -f ./test-file.tar.gz ./test-watcher.log
    
    echo -e "${GREEN}Cleanup complete${NC}"
}

# Setup trap for cleanup
trap cleanup EXIT INT TERM

# Test result functions
pass_test() {
    echo -e "${GREEN}✓ PASS:${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

fail_test() {
    echo -e "${RED}✗ FAIL:${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

# Main test execution
echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}  LibreSeed File Watcher Test Suite${NC}"
echo -e "${BLUE}======================================${NC}\n"

# Test 0: Prerequisites
echo -e "${YELLOW}[Test 0] Checking prerequisites...${NC}"

if [ ! -f "./build/seeder" ]; then
    info "Building seeder..."
    make build
    if [ $? -eq 0 ]; then
        pass_test "Seeder built successfully"
    else
        fail_test "Failed to build seeder"
        exit 1
    fi
else
    pass_test "Seeder binary exists"
fi

# Prepare test package
if [ -f "../test-package/hello-world@1.0.0.tgz" ]; then
    cp "../test-package/hello-world@1.0.0.tgz" "./test-file.tar.gz"
    pass_test "Test package prepared"
else
    fail_test "Test package not found at ../test-package/hello-world@1.0.0.tgz"
    exit 1
fi

# Create watch directories
mkdir -p ./packages/seeded ./packages/invalid
pass_test "Watch directories created"

# Test 1: Start seeder with watcher
echo -e "\n${YELLOW}[Test 1] Starting seeder with file watcher...${NC}"

./build/seeder start > test-watcher.log 2>&1 &
SEEDER_PID=$!
echo "$SEEDER_PID" > seeder.pid

sleep 3  # Wait for startup

if kill -0 "$SEEDER_PID" 2>/dev/null; then
    pass_test "Seeder started (PID: $SEEDER_PID)"
else
    fail_test "Seeder failed to start"
    cat test-watcher.log
    exit 1
fi

# Check for watcher initialization
if grep -q "File watcher started successfully" test-watcher.log; then
    pass_test "File watcher initialized"
else
    fail_test "File watcher not initialized"
    cat test-watcher.log
fi

# Test 2: Automatic package detection
echo -e "\n${YELLOW}[Test 2] Testing automatic package detection...${NC}"

info "Copying test package to watch directory..."
cp test-file.tar.gz ./packages/test-auto-1.tar.gz

# Wait for processing (2s debounce + processing time)
sleep 5

if grep -q "Processing package.*test-auto-1.tar.gz" test-watcher.log; then
    pass_test "Package detected by watcher"
else
    fail_test "Package not detected"
    cat test-watcher.log
fi

if grep -q "Successfully added package.*test-auto-1.tar.gz" test-watcher.log; then
    pass_test "Package added to seeder"
else
    fail_test "Package not added to seeder"
fi

# Test 3: File movement
echo -e "\n${YELLOW}[Test 3] Testing file movement...${NC}"

if [ -f "./packages/seeded/test-auto-1.tar.gz" ]; then
    pass_test "Successfully processed file moved to seeded/"
else
    fail_test "File not moved to seeded/"
    ls -la ./packages/
fi

if [ ! -f "./packages/test-auto-1.tar.gz" ]; then
    pass_test "Original file removed from watch directory"
else
    fail_test "Original file still in watch directory"
fi

# Test 4: Invalid package handling
echo -e "\n${YELLOW}[Test 4] Testing invalid package handling...${NC}"

echo "This is not a valid tarball" > ./packages/invalid-test.tar.gz

sleep 5

if grep -q "Failed to add package.*invalid-test.tar.gz" test-watcher.log; then
    pass_test "Invalid package detected as error"
else
    fail_test "Invalid package not detected as error"
fi

if [ -f "./packages/invalid/invalid-test.tar.gz" ]; then
    pass_test "Invalid file moved to invalid/"
else
    fail_test "Invalid file not moved to invalid/"
    ls -la ./packages/
fi

# Test 5: Multiple files
echo -e "\n${YELLOW}[Test 5] Testing multiple file processing...${NC}"

info "Adding 3 files simultaneously..."
for i in {1..3}; do
    cp test-file.tar.gz "./packages/test-multi-$i.tar.gz" &
done
wait

sleep 8  # Wait for all to process

PROCESSED_COUNT=$(ls -1 ./packages/seeded/test-multi-*.tar.gz 2>/dev/null | wc -l)
if [ "$PROCESSED_COUNT" -eq 3 ]; then
    pass_test "All 3 files processed ($PROCESSED_COUNT/3)"
else
    fail_test "Not all files processed ($PROCESSED_COUNT/3)"
fi

# Test 6: Seeder status verification
echo -e "\n${YELLOW}[Test 6] Verifying seeder status...${NC}"

# Give a moment for status to stabilize
sleep 2

# Note: The seeder status command may not work while seeder is running in background
# This is a known limitation - we'll check logs instead
ADDED_COUNT=$(grep -c "Successfully added package" test-watcher.log || echo "0")
if [ "$ADDED_COUNT" -ge 4 ]; then
    pass_test "Multiple packages added (count: $ADDED_COUNT)"
else
    fail_test "Expected 4+ packages added, found: $ADDED_COUNT"
fi

# Test 7: Graceful shutdown
echo -e "\n${YELLOW}[Test 7] Testing graceful shutdown...${NC}"

info "Sending SIGTERM to seeder..."
kill -TERM "$SEEDER_PID"

sleep 3

if ! kill -0 "$SEEDER_PID" 2>/dev/null; then
    pass_test "Seeder stopped gracefully"
else
    fail_test "Seeder still running after SIGTERM"
    kill -9 "$SEEDER_PID" 2>/dev/null || true
fi

if grep -q "Stopping file watcher" test-watcher.log; then
    pass_test "File watcher shutdown message logged"
else
    fail_test "File watcher shutdown not logged properly"
fi

# Final report
echo -e "\n${BLUE}======================================${NC}"
echo -e "${BLUE}           Test Summary${NC}"
echo -e "${BLUE}======================================${NC}"
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
echo -e "Total Tests:  $((TESTS_PASSED + TESTS_FAILED))"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}✓ All tests passed!${NC}\n"
    exit 0
else
    echo -e "\n${RED}✗ Some tests failed. See test-watcher.log for details.${NC}\n"
    echo -e "${YELLOW}Last 50 lines of log:${NC}"
    tail -n 50 test-watcher.log
    exit 1
fi
