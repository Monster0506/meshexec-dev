//go:build !linux && !darwin && !windows

package ble

import (
	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

// NewNativeTransport on unsupported platforms falls back to the simulated transport.
func NewNativeTransport(cfg *core.NetworkConfig) (core.BLETransport, error) {
	return createNativeFallback(cfg, nil)
}

// newNativeWithLogger on unsupported platforms falls back to the simulated transport with logging.
func newNativeWithLogger(cfg *core.NetworkConfig, logger *logging.Logger) (core.BLETransport, error) {
	return createNativeFallback(cfg, logger)
}
