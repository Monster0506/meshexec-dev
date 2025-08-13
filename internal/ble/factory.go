//go:build ble

package ble

import (
	"fmt"
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
	return nil, fmt.Errorf("BLE disabled in this build")
}

/*
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
	case "sidecar":
		if t, ok, err := tryNewSidecarTransport(cfg, logger); ok {
			if err == nil {
				logger.Info("Using Windows sidecar BLE transport", nil)
				return t, nil
			}
			logger.Error("Sidecar BLE transport initialization failed", err, nil)
			return nil, err
		}
		logger.Error("Sidecar transport not available in this build", nil, nil)
		return nil, fmt.Errorf("sidecar transport not available")
	case "native", "goble", "tinygo", "winrt":
		logger.Info("Attempting native BLE transport (forced by environment)", nil)
		// Prefer sidecar on Windows when available
		if runtime.GOOS == "windows" {
			if t, ok, err := tryNewSidecarTransport(cfg, logger); ok {
				if err == nil {
					return t, nil
				}
				logger.Warn("Sidecar transport failed; continuing to native backends", map[string]interface{}{"error": err.Error()})
			}
		}
		// Prefer WinRT transport when available (Windows + build tag winrt)
		if t, ok, err := tryNewWinRT(cfg, logger); ok {
			if err == nil {
				return t, nil
			}
			// If requested impl was explicitly winrt, return error immediately
			if implType == "winrt" {
				logger.Error("WinRT BLE transport initialization failed", err, map[string]interface{}{"platform": getPlatform()})
				return nil, err
			}
			logger.Warn("WinRT transport failed; falling back to other native", map[string]interface{}{"error": err.Error()})
		}
		if t, err := newNativeWithLogger(cfg, logger); err == nil {
			return t, nil
		} else {
			// Do not fall back to simulation on Windows when native is requested
			if runtime.GOOS == "windows" {
				logger.Error("Native BLE failed and simulation fallback is disabled on Windows", err, map[string]interface{}{
					"platform": getPlatform(),
				})
				return nil, err
			}
			logger.Warn("Native BLE failed, falling back to simulation", map[string]interface{}{
				"error":    err.Error(),
				"platform": getPlatform(),
			})
		}
		// If native init fails, fall back to sim
		return newSim(cfg, logger), nil
	default:
		logger.Info("Auto-detecting BLE transport capability", nil)
		// On Windows, try sidecar first
		if runtime.GOOS == "windows" {
			if t, ok, err := tryNewSidecarTransport(cfg, logger); ok {
				if err == nil {
					logger.Info("Using Windows sidecar BLE transport (auto)", nil)
					return t, nil
				}
				logger.Warn("Sidecar transport unavailable; trying WinRT/native", map[string]interface{}{"error": err.Error()})
			}
		}
		// Prefer WinRT when available
		if t, ok, err := tryNewWinRT(cfg, logger); ok {
			if err == nil {
				logger.Info("Using WinRT BLE transport", map[string]interface{}{"platform": getPlatform()})
				return t, nil
			}
			// If WinRT present but failed, log and continue to other natives
			logger.Warn("WinRT transport unavailable; trying other native backends", map[string]interface{}{"error": err.Error()})
		}
		if t, err := newNativeWithLogger(cfg, logger); err == nil {
			logger.Info("Using native BLE transport", map[string]interface{}{
				"platform":       getPlatform(),
				"transport_type": getActualTransportType(),
			})
			return t, nil
		} else {
			// Do not fall back to simulation on Windows in auto mode
			if runtime.GOOS == "windows" {
				logger.Error("Native BLE unavailable and simulation fallback is disabled on Windows", err, map[string]interface{}{
					"platform": getPlatform(),
				})
				return nil, err
			}
			logger.Info("Native BLE unavailable, using simulation", map[string]interface{}{
				"native_error": err.Error(),
				"platform":     getPlatform(),
			})
		}
		return newSim(cfg, logger), nil
	}
}
*/

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

// (removed) createNativeFallback was unused; simulation fallback is handled directly via newSim
