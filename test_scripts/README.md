# MeshExec BLE Testing Scripts

This directory contains comprehensive testing scripts for MeshExec's BLE functionality across all supported platforms.

## 🎉 **Windows BLE Parity Achievement**

**Windows now has identical BLE functionality to Unix systems!** All testing scripts have been updated to reflect the new Windows hybrid BLE implementation.

## Available Scripts

### Platform-Specific Tests

#### `test_ble_linux.sh` - Linux/macOS Testing Suite
- **Purpose**: Comprehensive BLE testing for Linux and macOS systems
- **Features**:
  - Simulated BLE testing
  - Native BLE testing (go-ble library)
  - Configuration file testing
  - Performance and reliability testing
  - Error handling validation
  - Multi-device simulation
  - Logging verification
  - Platform-specific checks
  - Fallback behavior testing
  - JSON output validation

#### `test_ble_simple.sh` - Simple Linux/macOS Testing
- **Purpose**: Basic BLE testing for Linux and macOS systems (recommended for quick testing)
- **Features**:
  - Simulated BLE discovery
  - Native BLE discovery
  - Mesh join functionality
  - Configuration testing
  - Error handling validation
  - Simplified test flow

**Usage**:
```bash
# Make executable
chmod +x test_ble_linux.sh
chmod +x test_ble_simple.sh

# Run comprehensive tests from cmd/meshexec directory
./test_ble_linux.sh

# Run simple tests from cmd/meshexec directory (recommended)
./test_ble_simple.sh
```

#### `test_ble_windows.ps1` - Windows PowerShell Testing Suite
- **Purpose**: Comprehensive BLE testing for Windows systems
- **Features**:
  - Simulated BLE testing
  - **Native Windows BLE testing** (NEW - Hybrid TinyGo implementation)
  - Configuration file testing
  - Performance and reliability testing
  - Error handling validation
  - Multi-device simulation
  - Logging verification
  - Windows-specific platform checks
  - Fallback behavior testing
  - JSON output validation
  - **Hybrid transport verification** (NEW)
  - **Windows native BLE capability testing** (NEW)

#### `test_ble_simple.ps1` - Simple Windows PowerShell Testing
- **Purpose**: Basic BLE testing for Windows systems (recommended for quick testing)
- **Features**:
  - Simulated BLE discovery
  - Native BLE discovery
  - Mesh join functionality
  - Configuration testing
  - Error handling validation
  - Simplified test flow

**Usage**:
```powershell
# Run comprehensive tests from cmd/meshexec directory
powershell -ExecutionPolicy Bypass -File test_ble_windows.ps1

# Run simple tests from cmd/meshexec directory (recommended)
powershell -ExecutionPolicy Bypass -File test_ble_simple.ps1

# With verbose output
powershell -ExecutionPolicy Bypass -File test_ble_windows.ps1 -Verbose
```

### Cross-Platform Tests

#### `test_ble_cross_platform.sh` - Cross-Platform Testing Suite
- **Purpose**: Tests BLE functionality across multiple devices and platforms
- **Features**:
  - Platform detection and validation
  - Single device discovery testing
  - Multi-device mesh formation testing
  - Cross-platform compatibility verification
  - Performance and reliability testing
  - Error recovery testing
  - Platform-specific capability checks
  - Test report generation

**Usage**:
```bash
# Make executable
chmod +x test_ble_cross_platform.sh

# Run from cmd/meshexec directory
./test_ble_cross_platform.sh
```

### Test Runner

#### `run_all_tests.sh` - Universal Test Runner
- **Purpose**: Automatically runs all available tests for the current platform
- **Features**:
  - Platform detection
  - Automatic script selection
  - Comprehensive test execution
  - Summary reporting

**Usage**:
```bash
# Make executable
chmod +x run_all_tests.sh

# Run from any directory
./run_all_tests.sh
```

## Prerequisites

### All Platforms
- MeshExec binary (`meshexec` or `meshexec.exe`) in the `cmd/meshexec` directory
- `jq` command-line JSON processor (for cross-platform tests)

### Linux/macOS
- Bash shell
- `timeout` command (usually available by default)
- Bluetooth permissions (may require `sudo` on Linux)

### Windows
- PowerShell 5.1 or later
- Execution policy allowing script execution
- Bluetooth hardware (for native BLE testing)

## Installation

### Installing jq (Required for Cross-Platform Tests)

**Linux (Ubuntu/Debian)**:
```bash
sudo apt-get install jq
```

**macOS**:
```bash
brew install jq
```

**Windows**:
```powershell
# Using Chocolatey
choco install jq

# Using Scoop
scoop install jq
```

## Test Scenarios

### 1. Basic Functionality Testing
- Simulated BLE transport initialization
- Device discovery (simulated and native)
- Mesh join operations
- Configuration file loading

### 2. Native BLE Testing
- Real BLE device discovery
- Hardware adapter validation
- Permission checks
- Performance characteristics

### 3. Multi-Device Testing
- Background advertising
- Cross-device discovery
- Mesh network formation
- Peer communication

### 4. Error Handling
- Invalid parameters
- Timeout scenarios
- Fallback mechanisms
- Recovery procedures

### 5. Performance Testing
- High-frequency scanning
- Memory usage
- CPU utilization
- Network stability

### 6. Cross-Platform Compatibility
- Platform-specific implementations
- Feature parity validation
- Interoperability testing
- Configuration compatibility

## Environment Variables

The test scripts use the following environment variables:

### BLE Implementation Control
- `MESHEXEC_BLE_IMPL`: Controls BLE transport type
  - `native`: Use real BLE hardware
  - `sim`: Use simulated transport
  - `invalid`: Test fallback behavior

### Logging Configuration
- `MESHEXEC_LOGGING_LEVEL`: Log verbosity (`debug`, `info`, `warn`, `error`)
- `MESHEXEC_LOGGING_FORMAT`: Log format (`json`, `text`)

### Device Configuration
- `MESHEXEC_DEVICE_NAME`: Device name for testing
- `MESHEXEC_CONFIG_FILE`: Configuration file path

## Test Reports

The cross-platform test script generates JSON test reports:
```
ble_test_report_<platform>_<timestamp>.json
```

Example report structure:
```json
{
  "test_report": {
    "platform": "windows",
    "timestamp": "2024-12-19T10:30:00Z",
    "test_suite": "MeshExec Cross-Platform BLE Testing",
    "results": {
      "single_device_discovery": "completed",
      "multi_device_mesh": "completed",
      "cross_platform_compatibility": "completed",
      "performance_reliability": "completed",
      "error_recovery": "completed"
    },
    "summary": "All cross-platform BLE tests completed successfully"
  }
}
```

## Troubleshooting

### Common Issues

**Permission Denied (Linux)**:
```bash
# Make scripts executable
chmod +x *.sh

# Run with sudo if needed for BLE access
sudo ./test_ble_linux.sh
```

**PowerShell Execution Policy (Windows)**:
```powershell
# Set execution policy
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser

# Or run with bypass
powershell -ExecutionPolicy Bypass -File test_ble_windows.ps1
```

**jq Not Found**:
```bash
# Install jq (see Installation section above)
# Or skip cross-platform tests if not needed
```

**MeshExec Binary Not Found**:
```bash
# Ensure you're running from cmd/meshexec directory
# Or build the binary first
go build -o meshexec .
```

### Platform-Specific Issues

**Linux Bluetooth Access**:
```bash
# Check Bluetooth service
sudo systemctl status bluetooth

# Check device permissions
ls -la /dev/bluetooth

# Add user to bluetooth group
sudo usermod -a -G bluetooth $USER
```

**macOS Bluetooth Permissions**:
- Ensure Bluetooth permissions are granted in System Preferences
- Check if Bluetooth is enabled
- Verify code signing if applicable

**Windows BLE Capabilities**:
- Ensure Bluetooth is enabled in Windows Settings
- Check device manager for Bluetooth adapters
- Verify TinyGo dependencies are installed

## Success Criteria

A successful test run should show:
- ✅ All test scenarios completed
- ✅ Platform-specific functionality verified
- ✅ Cross-platform compatibility confirmed
- ✅ Error handling mechanisms validated
- ✅ Performance characteristics acceptable
- ✅ Test reports generated successfully

## Contributing

When adding new test scenarios:
1. Update the appropriate platform-specific script
2. Add corresponding tests to cross-platform script
3. Update this README with new features
4. Ensure backward compatibility
5. Test on all supported platforms

## Version History

- **v1.0**: Initial test scripts with basic BLE functionality
- **v2.0**: Added Windows PowerShell testing suite
- **v3.0**: Enhanced cross-platform compatibility testing
- **v4.0**: **Windows BLE parity implementation** - Full feature parity achieved
- **v4.1**: Added hybrid transport verification and native Windows BLE testing

---

**Last Updated**: December 2024  
**Status**: ✅ All platforms have full BLE parity  
**Next Milestone**: Performance optimization and additional test scenarios
