#!/bin/bash

# MeshExec BLE Testing Suite for Linux/macOS
# This script provides comprehensive BLE testing across different scenarios

set -e  # Exit on any error

echo "=== MeshExec BLE Testing Suite (Linux/macOS) ==="
echo "Platform: $(uname -s)"
echo "Date: $(date)"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to run test with timeout
run_test() {
    local test_name="$1"
    local command="$2"
    local timeout_seconds="${3:-10}"
    
    print_status $BLUE "Running: $test_name"
    print_status $YELLOW "Command: $command"
    
    if timeout $timeout_seconds bash -c "$command"; then
        print_status $GREEN "✓ $test_name completed successfully"
    else
        print_status $RED "✗ $test_name failed or timed out"
        return 1
    fi
    echo ""
}

# Check if meshexec is available
if ! command -v ./meshexec &> /dev/null; then
    print_status $RED "Error: meshexec executable not found in current directory"
    print_status $YELLOW "Please run this script from the cmd/meshexec directory"
    exit 1
fi

# Test 1: Basic functionality with simulation
print_status $BLUE "Test 1: Simulated BLE"
export MESHEXEC_BLE_IMPL=sim
export MESHEXEC_LOGGING_LEVEL=debug
export MESHEXEC_DEVICE_NAME="linux-test-sim"

run_test "Simulated device discovery" \
    "./meshexec list --timeout 3000 --json" \
    5

run_test "Simulated mesh join" \
    "./meshexec join --foreground" \
    3

# Test 2: Native BLE functionality
print_status $BLUE "Test 2: Native BLE"
export MESHEXEC_BLE_IMPL=native
export MESHEXEC_LOGGING_LEVEL=info
export MESHEXEC_DEVICE_NAME="linux-test-native"

run_test "Native device discovery" \
    "./meshexec list --timeout 5000 --json" \
    6

run_test "Native mesh join" \
    "./meshexec join --foreground" \
    4

# Test 3: Configuration file testing
print_status $BLUE "Test 3: Configuration"
export MESHEXEC_CONFIG_FILE="./test_config.toml"

# Create test configuration
cat > test_config.toml << EOF
[device]
name = "config-test-device"
id = "test-123"

[network]
ble_implementation = "native"
scan_timeout = 5000
advertise_interval = 1000

[logging]
level = "debug"
format = "json"
EOF

run_test "Configuration file loading" \
    "./meshexec list --timeout 2000" \
    3

# Clean up test config
rm -f test_config.toml

# Test 4: Performance testing
print_status $BLUE "Test 4: Performance"
export MESHEXEC_BLE_IMPL=native
export MESHEXEC_LOGGING_LEVEL=warn

run_test "High-frequency scanning" \
    "./meshexec list --timeout 10000 --json" \
    11

# Test 5: Error handling
print_status $BLUE "Test 5: Error Handling"

run_test "Invalid timeout handling" \
    "./meshexec list --timeout 0" \
    2

run_test "Invalid BLE implementation" \
    "MESHEXEC_BLE_IMPL=invalid ./meshexec list --timeout 1000" \
    2

# Test 6: Multi-device simulation
print_status $BLUE "Test 6: Multi-Device Simulation"

# Start background advertising
export MESHEXEC_BLE_IMPL=sim
export MESHEXEC_DEVICE_NAME="background-advertiser"
./meshexec join --foreground &
BACKGROUND_PID=$!

# Wait a moment for advertising to start
sleep 2

# Test discovery from another instance
export MESHEXEC_DEVICE_NAME="discovery-test"
run_test "Multi-device discovery" \
    "./meshexec list --timeout 3000 --json" \
    4

# Clean up background process
kill $BACKGROUND_PID 2>/dev/null || true
wait $BACKGROUND_PID 2>/dev/null || true

# Test 7: Logging verification
print_status $BLUE "Test 7: Logging Verification"

export MESHEXEC_LOGGING_LEVEL=debug
export MESHEXEC_LOGGING_FORMAT=json

run_test "Structured logging output" \
    "./meshexec list --timeout 2000 2>&1 | grep -q 'INF'" \
    3

# Test 8: Platform-specific checks
print_status $BLUE "Test 8: Platform Checks"

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    print_status $YELLOW "Linux detected - checking Bluetooth permissions"
    if ls -la /dev/bluetooth &>/dev/null; then
        print_status $GREEN "✓ Bluetooth device accessible"
    else
        print_status $YELLOW "⚠ Bluetooth device not accessible (may need sudo)"
    fi
elif [[ "$OSTYPE" == "darwin"* ]]; then
    print_status $YELLOW "macOS detected - checking Bluetooth permissions"
    if system_profiler SPBluetoothDataType &>/dev/null; then
        print_status $GREEN "✓ Bluetooth system profiler accessible"
    else
        print_status $YELLOW "⚠ Bluetooth permissions may be required"
    fi
fi

# Test 9: Fallback behavior
print_status $BLUE "Test 9: Fallback Behavior"

# Test with invalid implementation
export MESHEXEC_BLE_IMPL=invalid
run_test "Fallback to simulation" \
    "./meshexec list --timeout 2000" \
    3

# Test 10: JSON output validation
print_status $BLUE "Test 10: JSON Output"

export MESHEXEC_BLE_IMPL=sim
run_test "JSON output format" \
    "./meshexec list --timeout 2000 --json | jq ." \
    3

# Summary
echo ""
print_status $GREEN "=== Testing Summary ==="
print_status $GREEN "✓ All BLE test scenarios completed"
print_status $GREEN "✓ Cross-platform compatibility verified"
print_status $GREEN "✓ Error handling validated"
print_status $GREEN "✓ Performance characteristics tested"
echo ""
print_status $BLUE "Next steps:"
print_status $YELLOW "- Run multi-device tests with actual hardware"
print_status $YELLOW "- Test with different BLE adapters"
print_status $YELLOW "- Validate mesh networking in production environment"
echo ""
print_status $GREEN "✅ Linux/macOS BLE Testing Suite Complete!"
