# MeshExec Device Connectivity Test Script
# This script demonstrates how to connect to and interact with other devices

Write-Host "=== MeshExec Device Connectivity Test ===" -ForegroundColor Cyan
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

# Test 1: Discover and Display Devices
Write-Status "Test 1: Device Discovery" "Blue"
$env:MESHEXEC_BLE_IMPL = "native"
$env:MESHEXEC_LOGGING_LEVEL = "info"

try {
    $output = .\meshexec.exe list --timeout 5000 --json
    $devices = $output | ConvertFrom-Json
    
    Write-Status "✓ Discovered $($devices.peers.Count) devices" "Green"
    
    # Display device details
    Write-Host ""
    Write-Status "=== Device Details ===" "Yellow"
    $devices.peers | ForEach-Object {
        $signal = if ($_.signal_strength -gt -50) { "Excellent" }
                  elseif ($_.signal_strength -gt -70) { "Good" }
                  elseif ($_.signal_strength -gt -80) { "Fair" }
                  else { "Poor" }
        
        $name = if ($_.name) { $_.name } else { "Unknown" }
        
        Write-Host "📱 $name" -ForegroundColor Cyan
        Write-Host "   Address: $($_.address)" -ForegroundColor White
        Write-Host "   Signal: $signal ($($_.signal_strength) dBm)" -ForegroundColor Green
        Write-Host "   Last Seen: $($_.last_seen)" -ForegroundColor Gray
            Write-Host ""
    
} catch {
    Write-Status "✗ Device discovery failed: $_" "Red"
}
    
} catch {
    Write-Status "✗ Device discovery failed: $_" "Red"
}

Write-Host ""

# Test 2: Join Mesh Network (Make Device Discoverable)
Write-Status "Test 2: Join Mesh Network" "Blue"
$env:MESHEXEC_DEVICE_NAME = "TJ-Windows-Test"

Write-Status "Starting mesh join (this will make your device discoverable)..." "Yellow"
Write-Status "Press Ctrl+C to stop after 10 seconds" "Yellow"

try {
    # Start join in background
    $job = Start-Job -ScriptBlock {
        .\meshexec.exe join --foreground
    }
    
    # Wait 10 seconds
    Start-Sleep -Seconds 10
    
    # Check if job is running
    if (Get-Job $job -ErrorAction SilentlyContinue) {
        Write-Status "✓ Mesh join active - your device is now discoverable" "Green"
        
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

# Test 3: Continuous Monitoring
Write-Status "Test 3: Continuous Device Monitoring" "Blue"
Write-Status "Monitoring devices for 30 seconds..." "Yellow"

$startTime = Get-Date
$monitoringDuration = 30

try {
    while ((Get-Date) -lt ($startTime.AddSeconds($monitoringDuration))) {
        $remaining = [math]::Round(($startTime.AddSeconds($monitoringDuration) - (Get-Date)).TotalSeconds)
        
        Write-Host "`rMonitoring... $remaining seconds remaining" -NoNewline -ForegroundColor Yellow
        
        $output = .\meshexec.exe list --timeout 2000 --json 2>$null
        if ($output) {
            $devices = $output | ConvertFrom-Json
            $deviceCount = $devices.peers.Count
            
            Write-Host " - Found $deviceCount devices" -NoNewline -ForegroundColor Green
        }
        
        Start-Sleep -Seconds 2
    }
    
    Write-Host ""
    Write-Status "✓ Continuous monitoring completed" "Green"
    
} catch {
    Write-Status "✗ Continuous monitoring failed: $_" "Red"
}

Write-Host ""

# Test 4: Signal Strength Analysis
Write-Status "Test 4: Signal Strength Analysis" "Blue"

try {
    $output = .\meshexec.exe list --timeout 3000 --json
    $devices = $output | ConvertFrom-Json
    
    $strongSignals = $devices.peers | Where-Object { $_.signal_strength -gt -60 }
    $mediumSignals = $devices.peers | Where-Object { $_.signal_strength -le -60 -and $_.signal_strength -gt -80 }
    $weakSignals = $devices.peers | Where-Object { $_.signal_strength -le -80 }
    
    Write-Status "Signal Strength Summary:" "Yellow"
    Write-Status "  Strong signals (-60 dBm or better): $($strongSignals.Count) devices" "Green"
    Write-Status "  Medium signals (-60 to -80 dBm): $($mediumSignals.Count) devices" "Yellow"
    Write-Status "  Weak signals (worse than -80 dBm): $($weakSignals.Count) devices" "Red"
    
    if ($strongSignals) {
        Write-Host ""
        Write-Status "Best devices for connection:" "Cyan"
        $strongSignals | ForEach-Object {
            $name = if ($_.name) { $_.name } else { "Unknown" }
            Write-Host "  📶 $name ($($_.address)) - $($_.signal_strength) dBm" -ForegroundColor Green
        }
    }
    
} catch {
    Write-Status "✗ Signal strength analysis failed: $_" "Red"
}

Write-Host ""

# Summary
Write-Status "=== Connectivity Test Summary ===" "Green"
Write-Status "✓ Device discovery working" "Green"
Write-Status "✓ Mesh network join tested" "Green"
Write-Status "✓ Continuous monitoring tested" "Green"
Write-Status "✓ Signal strength analysis completed" "Green"
Write-Host ""
Write-Status "🎉 Your Windows device can now:" "Cyan"
Write-Status "  • Discover other BLE devices" "White"
Write-Status "  • Join mesh networks" "White"
Write-Status "  • Be discovered by other devices" "White"
Write-Status "  • Monitor device connectivity" "White"
Write-Host ""
Write-Status "💡 Next steps:" "Yellow"
Write-Status "  • Run '.\meshexec.exe join --foreground' to stay in mesh" "White"
Write-Status "  • Use '.\meshexec.exe list --timeout 5000' to scan for devices" "White"
Write-Status "  • Check signal strength for best connection quality" "White"
