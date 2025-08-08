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
// MECHEXEC_BLE_IMPL with values: "native" or "sim".
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
    
    implType := strings.ToLower(strings.TrimSpace(os.Getenv("MECHEXEC_BLE_IMPL")))
    logger.Info("Creating BLE transport", map[string]interface{}{
        "requested_impl": func() string {
            if implType == "" {
                return "auto"
            }
            return implType
        }(),
    })
    
    switch implType {
    case "sim", "simulator", "mock":
        logger.Info("Using simulated BLE transport (forced by environment)", nil)
        return newSim(cfg, logger), nil
    case "native", "goble":
        logger.Info("Attempting native BLE transport (forced by environment)", nil)
        if t, err := newNativeWithLogger(cfg, logger); err == nil {
            return t, nil
        } else {
            logger.Warn("Native BLE failed, falling back to simulation", map[string]interface{}{
                "error": err.Error(),
            })
        }
        // If native init fails, fall back to sim
        return newSim(cfg, logger), nil
    default:
        logger.Info("Auto-detecting BLE transport capability", nil)
        if t, err := newNativeWithLogger(cfg, logger); err == nil {
            logger.Info("Using native BLE transport", nil)
            return t, nil
        } else {
            logger.Info("Native BLE unavailable, using simulation", map[string]interface{}{
                "native_error": err.Error(),
            })
        }
        return newSim(cfg, logger), nil
    }
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

