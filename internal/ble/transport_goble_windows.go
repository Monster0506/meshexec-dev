//go:build windows && goble

package ble

import (
    core "github.com/monster0506/meshexec/internal"
    "github.com/monster0506/meshexec/internal/logging"
)

// NewNativeTransport on Windows falls back to the simulated transport for now.
func NewNativeTransport(_ *core.NetworkConfig) (core.BLETransport, error) {
    return NewTransport(), nil
}

// newNativeWithLogger on Windows falls back to the simulated transport with logging.
func newNativeWithLogger(cfg *core.NetworkConfig, logger *logging.Logger) (core.BLETransport, error) {
    if logger != nil {
        logger.Info("Windows native BLE not implemented, using simulation", nil)
    }
    return NewTransportWithLogger(logger), nil
}

