//go:build darwin

package ble

import (
    goble "github.com/go-ble/ble"
    "github.com/go-ble/ble/darwin"
)

func newDevice() (goble.Device, error) {
    return darwin.NewDevice()
}

