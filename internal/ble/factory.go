package ble

import (
	"os"
	"strconv"
	"strings"
	"time"

	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

// New returns a BLETransport using native BLE when available, falling back to a
// simulator. The behavior can be forced via the environment variable
// MESHEXEC_BLE_IMPL with values: "native" or "sim".
//
// If cfg is provided, certain simulation parameters (e.g., advertise interval)
// are applied to improve test fidelity.
func New(cfg *core.NetworkConfig) (core.BLETransport, error) {
	return NewWithLogger(cfg, nil)
}

// NewWithLogger returns a BLETransport with logging support.
func NewWithLogger(cfg *core.NetworkConfig, logger *logging.Logger) (core.BLETransport, error) {
	if logger == nil {
    	logger = logging.NewLogger("info")
	}

	implType := strings.ToLower(strings.TrimSpace(os.Getenv("MESHEXEC_BLE_IMPL")))
	logger.Info("Creating BLE transport", map[string]interface{}{
		"requested_impl": func() string {
			if implType == "" {
				return "auto"
			}
			return implType
		}(),
		"platform": getPlatform(),
	})

	switch implType {
	case "sim", "simulator", "mock":
		logger.Info("Using simulated BLE transport (forced by environment)", nil)
		return newSim(cfg, logger), nil
	case "native", "goble", "tinygo":
		logger.Info("Attempting native BLE transport (forced by environment)", nil)
		if t, err := newNativeWithLogger(cfg, logger); err == nil {
			return t, nil
		} else {
			logger.Warn("Native BLE failed, falling back to simulation", map[string]interface{}{
				"error":    err.Error(),
				"platform": getPlatform(),
			})
		}
		// If native init fails, fall back to sim
		return newSim(cfg, logger), nil
	default:
		logger.Info("Auto-detecting BLE transport capability", nil)
		if t, err := newNativeWithLogger(cfg, logger); err == nil {
			logger.Info("Using native BLE transport", map[string]interface{}{
				"platform":       getPlatform(),
				"transport_type": getActualTransportType(),
			})
			return t, nil
		} else {
			logger.Info("Native BLE unavailable, using simulation", map[string]interface{}{
				"native_error": err.Error(),
				"platform":     getPlatform(),
			})
		}
		return newSim(cfg, logger), nil
	}
}

// getPlatform returns the current platform for logging
func getPlatform() string {
	switch {
	case strings.Contains(strings.ToLower(os.Getenv("OS")), "windows"):
		return "windows"
	case fileExists("/proc/version"):
		return "linux"
	case fileExists("/System/Library/CoreServices/SystemVersion.plist"):
		return "darwin"
	default:
		return "unknown"
	}
}

// getActualTransportType returns what transport implementation is actually being used
func getActualTransportType() string {
	// This will be implemented differently per platform
	return "platform-native"
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func newSim(cfg *core.NetworkConfig, logger *logging.Logger) core.BLETransport {
	t := NewTransportWithLogger(logger)
	if cfg != nil {
		// Apply advertise interval if provided (milliseconds in config)
		if cfg.AdvertiseInterval > 0 {
			if tt, err := strconv.Atoi(strconv.Itoa(cfg.AdvertiseInterval)); err == nil {
				// cfg.AdvertiseInterval is already int; converting to duration
				d := time.Duration(tt) * time.Millisecond
				if d > 0 {
					// direct field access is allowed inside package ble
					t.advertiseInterval = d
					logger.Debug("Applied custom advertise interval to simulated transport", map[string]interface{}{
						"interval_ms": tt,
					})
				}
			}
		}
	}
	return t
}

// For backward compatibility - the old function signature without logger
func newSimCompat(cfg *core.NetworkConfig) core.BLETransport {
	return newSim(cfg, nil)
}

// Fallback for platforms without native BLE support
func createNativeFallback(cfg *core.NetworkConfig, logger *logging.Logger) (core.BLETransport, error) {
	if logger != nil {
		logger.Info("Native BLE not available on this platform, using simulation", nil)
	}
	return newSim(cfg, logger), nil
}
