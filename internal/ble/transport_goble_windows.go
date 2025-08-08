//go:build windows && goble

package ble

import (
    core "github.com/monster0506/mechexec/internal"
)

// NewNativeTransport on Windows falls back to the simulated transport for now.
func NewNativeTransport(_ *core.NetworkConfig) (core.BLETransport, error) {
    return NewTransport(), nil
}

