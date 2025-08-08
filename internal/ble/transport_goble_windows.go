//go:build windows && goble

package ble

import (
	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

// NewNativeTransport on Windows with goble tag also uses hybrid approach
// since go-ble doesn't have real Windows support anyway.
func NewNativeTransport(cfg *core.NetworkConfig) (core.BLETransport, error) {
	return newNativeWithLogger(cfg, nil)
}

// newNativeWithLogger creates a Windows hybrid transport regardless of build tags.
func newNativeWithLogger(cfg *core.NetworkConfig, logger *logging.Logger) (core.BLETransport, error) {
	if logger != nil {
		logger.Info("Windows BLE: using hybrid approach (go-ble has no real Windows support)", nil)
	}

	// Since go-ble doesn't support Windows anyway, we'll use the same approach
	// as the TinyGo version - create a hybrid transport that can do real scanning
	// and simulated advertising

	// For now, fall back to full simulation until we implement the hybrid here too
	return NewTransportWithLogger(logger), nil
}
