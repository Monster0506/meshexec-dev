//go:build windows && winrt

package ble

import (
	"fmt"

	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

// tryNewWinRT is compiled only when building with -tags winrt on Windows.
// For now, return a detectable error so factory can report WinRT presence but failure.
func tryNewWinRT(cfg *core.NetworkConfig, logger *logging.Logger) (core.BLETransport, bool, error) {
	return nil, true, fmt.Errorf("winrt transport not fully implemented yet (API alignment pending)")
}
