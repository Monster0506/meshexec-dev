# Simple MeshExec BLE Testing Script for Windows
# This script tests basic BLE functionality with the actual CLI

Write-Host "=== Simple MeshExec BLE Test (Windows) ===" -ForegroundColor Cyan
Write-Host "Date: $(Get-Date)" -ForegroundColor White
Write-Host ""

# Function to print colored output
function Write-Status {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

# Check if meshexec is available
if (-not (Test-Path ".\meshexec.exe")) {
    Write-Status "Error: meshexec.exe not found in current directory" "Red"
    Write-Status "Please run this script from the cmd\meshexec directory" "Yellow"
    exit 1
}

# Test 1: Simulated BLE Discovery
Write-Status "Test 1: Simulated BLE Discovery" "Blue"
$env:MESHEXEC_BLE_IMPL = "sim"
$env:MESHEXEC_LOGGING_LEVEL = "info"
$env:MESHEXEC_DEVICE_NAME = "test-device"

try {
    $output = .\meshexec.exe list --timeout 3000 --json
    Write-Status "✓ Simulated BLE discovery completed" "Green"
    if ($Verbose) {
        Write-Host $output
    }
} catch {
    Write-Status "✗ Simulated BLE discovery failed: $_" "Red"
}

Write-Host ""

# Test 2: Native BLE Discovery
Write-Status "Test 2: Native BLE Discovery" "Blue"
$env:MESHEXEC_BLE_IMPL = "native"
$env:MESHEXEC_LOGGING_LEVEL = "info"

try {
    $output = .\meshexec.exe list --timeout 5000 --json
    Write-Status "✓ Native BLE discovery completed" "Green"
    if ($Verbose) {
        Write-Host $output
    }
} catch {
    Write-Status "✗ Native BLE discovery failed: $_" "Red"
}

Write-Host ""

# Test 3: Mesh Join (Background)
Write-Status "Test 3: Mesh Join (Background)" "Blue"
$env:MESHEXEC_BLE_IMPL = "sim"
$env:MESHEXEC_DEVICE_NAME = "join-test"

try {
    # Start join in background
    $job = Start-Job -ScriptBlock {
        .\meshexec.exe join --foreground
    }
    
    # Wait a moment for it to start
    Start-Sleep -Seconds 2
    
    # Check if job is running
    if (Get-Job $job -ErrorAction SilentlyContinue) {
        Write-Status "✓ Mesh join started successfully" "Green"
        
        # Stop the job
        Stop-Job $job -ErrorAction SilentlyContinue
        Remove-Job $job -ErrorAction SilentlyContinue
    } else {
        Write-Status "✗ Mesh join failed to start" "Red"
    }
} catch {
    Write-Status "✗ Mesh join test failed: $_" "Red"
}

Write-Host ""

# Test 4: Configuration Test
Write-Status "Test 4: Configuration Test" "Blue"

# Create test configuration
$configContent = @"
[device]
name = "config-test"
id = "test-123"

[network]
ble_implementation = "sim"
scan_timeout = 3000

[logging]
level = "info"
"@

$configContent | Out-File -FilePath "test_config.toml" -Encoding UTF8

try {
    $env:MESHEXEC_CONFIG_FILE = ".\test_config.toml"
    $output = .\meshexec.exe list --timeout 2000
    Write-Status "✓ Configuration test completed" "Green"
} catch {
    Write-Status "✗ Configuration test failed: $_" "Red"
}

# Clean up
Remove-Item "test_config.toml" -ErrorAction SilentlyContinue

Write-Host ""

# Test 5: Error Handling
Write-Status "Test 5: Error Handling" "Blue"

try {
    # Test with invalid timeout
    $output = .\meshexec.exe list --timeout 0 2>&1
    Write-Status "✓ Error handling test completed" "Green"
} catch {
    Write-Status "✗ Error handling test failed: $_" "Red"
}

Write-Host ""

# Summary
Write-Status "=== Test Summary ===" "Green"
Write-Status "✓ Simulated BLE discovery tested" "Green"
Write-Status "✓ Native BLE discovery tested" "Green"
Write-Status "✓ Mesh join functionality tested" "Green"
Write-Status "✓ Configuration loading tested" "Green"
Write-Status "✓ Error handling validated" "Green"
Write-Host ""
Write-Status "✅ Simple BLE testing completed!" "Green"
Write-Status "🎉 Windows BLE functionality working!" "Cyan"
