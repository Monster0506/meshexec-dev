//go:build !windows

package ble

import (
    core "github.com/monster0506/meshexec/internal"
    "github.com/monster0506/meshexec/internal/logging"
)

// tryNewSidecarTransport is unavailable on non-Windows builds.
// Return (nil, false, nil) to signal not supported so factory can skip it.
func tryNewSidecarTransport(cfg *core.NetworkConfig, logger *logging.Logger) (core.BLETransport, bool, error) {
    return nil, false, nil
}


