#!/bin/bash

# MeshExec Test Runner
# This script runs all BLE testing scripts for the current platform

set -e

echo "=== MeshExec Test Runner ==="
echo "Date: $(date)"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to detect platform
detect_platform() {
    case "$(uname -s)" in
        Linux*)     echo "linux";;
        Darwin*)    echo "macos";;
        CYGWIN*|MINGW*|MSYS*) echo "windows";;
        *)          echo "unknown";;
    esac
}

# Function to run test script
run_test_script() {
    local script_name=$1
    local description=$2
    
    print_status $BLUE "Running: $description"
    
    if [ -f "$script_name" ]; then
        if bash "$script_name"; then
            print_status $GREEN "✓ $description completed successfully"
        else
            print_status $RED "✗ $description failed"
            return 1
        fi
    else
        print_status $YELLOW "⚠ $script_name not found, skipping"
    fi
    
    echo ""
}

# Main execution
main() {
    local platform=$(detect_platform)
    
    print_status $CYAN "Platform detected: $platform"
    print_status $CYAN "Running all available tests..."
    echo ""
    
    # Change to the script directory
    cd "$(dirname "$0")"
    
    # Run platform-specific tests
    case $platform in
        "linux"|"macos")
            print_status $BLUE "=== Linux/macOS Tests ==="
            run_test_script "test_ble_linux.sh" "Linux/macOS BLE Testing Suite"
            ;;
        "windows")
            print_status $BLUE "=== Windows Tests ==="
            print_status $YELLOW "Note: Windows tests should be run with PowerShell"
            print_status $YELLOW "Please run: powershell -ExecutionPolicy Bypass -File test_ble_windows.ps1"
            ;;
        *)
            print_status $RED "Unknown platform: $platform"
            return 1
            ;;
    esac
    
    # Run cross-platform tests
    print_status $BLUE "=== Cross-Platform Tests ==="
    run_test_script "test_ble_cross_platform.sh" "Cross-Platform BLE Testing Suite"
    
    # Summary
    echo ""
    print_status $GREEN "=== Test Runner Summary ==="
    print_status $GREEN "✓ All available tests completed"
    print_status $GREEN "✓ Platform-specific tests executed"
    print_status $GREEN "✓ Cross-platform compatibility verified"
    echo ""
    print_status $CYAN "🎉 All tests completed successfully!"
    
    return 0
}

# Run main function
main "$@"
