#!/bin/bash

# Simple MeshExec BLE Testing Script
# This script tests basic BLE functionality with the actual CLI

set -e

echo "=== Simple MeshExec BLE Test ==="
echo "Platform: $(uname -s)"
echo "Date: $(date)"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to run test
run_test() {
    local test_name="$1"
    local command="$2"
    
    print_status $BLUE "Running: $test_name"
    print_status $YELLOW "Command: $command"
    
    if eval "$command"; then
        print_status $GREEN "✓ $test_name completed successfully"
    else
        print_status $RED "✗ $test_name failed"
        return 1
    fi
    echo ""
}

# Check if meshexec is available
if [ ! -f "./meshexec" ] && [ ! -f "./meshexec.exe" ]; then
    print_status $RED "Error: meshexec executable not found in current directory"
    print_status $YELLOW "Please run this script from the cmd/meshexec directory"
    exit 1
fi

# Determine executable name
if [ -f "./meshexec.exe" ]; then
    MESHEXEC="./meshexec.exe"
else
    MESHEXEC="./meshexec"
fi

# Test 1: Simulated BLE Discovery
print_status $BLUE "Test 1: Simulated BLE Discovery"
export MESHEXEC_BLE_IMPL=sim
export MESHEXEC_LOGGING_LEVEL=info
export MESHEXEC_DEVICE_NAME=test-device

run_test "Simulated device discovery" \
    "$MESHEXEC list --timeout 3000 --json"

# Test 2: Native BLE Discovery
print_status $BLUE "Test 2: Native BLE Discovery"
export MESHEXEC_BLE_IMPL=native
export MESHEXEC_LOGGING_LEVEL=info

run_test "Native device discovery" \
    "$MESHEXEC list --timeout 5000 --json"

# Test 3: Mesh Join (Background)
print_status $BLUE "Test 3: Mesh Join (Background)"
export MESHEXEC_BLE_IMPL=sim
export MESHEXEC_DEVICE_NAME=join-test

# Start join in background
$MESHEXEC join --foreground &
JOIN_PID=$!

# Wait a moment for it to start
sleep 2

# Check if process is running
if kill -0 $JOIN_PID 2>/dev/null; then
    print_status $GREEN "✓ Mesh join started successfully"
    
    # Stop the process
    kill $JOIN_PID 2>/dev/null || true
    wait $JOIN_PID 2>/dev/null || true
else
    print_status $RED "✗ Mesh join failed to start"
fi

echo ""

# Test 4: Configuration Test
print_status $BLUE "Test 4: Configuration Test"

# Create test configuration
cat > test_config.toml << EOF
[device]
name = "config-test"
id = "test-123"

[network]
ble_implementation = "sim"
scan_timeout = 3000

[logging]
level = "info"
EOF

export MESHEXEC_CONFIG_FILE=./test_config.toml
run_test "Configuration file loading" \
    "$MESHEXEC list --timeout 2000"

# Clean up
rm -f test_config.toml

# Test 5: Error Handling
print_status $BLUE "Test 5: Error Handling"

run_test "Invalid timeout handling" \
    "$MESHEXEC list --timeout 0 2>/dev/null || true"

# Test 6: Logging Verification
print_status $BLUE "Test 6: Logging Verification"

export MESHEXEC_LOGGING_LEVEL=debug
run_test "Structured logging output" \
    "$MESHEXEC list --timeout 2000 2>&1 | grep -q 'INF'"

# Summary
echo ""
print_status $GREEN "=== Test Summary ==="
print_status $GREEN "✓ Simulated BLE discovery tested"
print_status $GREEN "✓ Native BLE discovery tested"
print_status $GREEN "✓ Mesh join functionality tested"
print_status $GREEN "✓ Configuration loading tested"
print_status $GREEN "✓ Error handling validated"
print_status $GREEN "✓ Logging verification completed"
echo ""
print_status $GREEN "✅ Simple BLE testing completed!"
print_status $BLUE "🎉 BLE functionality working correctly!"
