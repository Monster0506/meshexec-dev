//go:build linux

package ble

import (
	goble "github.com/go-ble/ble"
	"github.com/go-ble/ble/linux"
)

func newDevice() (goble.Device, error) {
	return linux.NewDevice()
}
