package ble

import (
    "os"
    "strconv"
    "strings"
    "time"

    core "github.com/monster0506/mechexec/internal"
)

// New returns a BLETransport using native BLE when available, falling back to a
// simulator. The behavior can be forced via the environment variable
// MECHEXEC_BLE_IMPL with values: "native" or "sim".
//
// If cfg is provided, certain simulation parameters (e.g., advertise interval)
// are applied to improve test fidelity.
func New(cfg *core.NetworkConfig) (core.BLETransport, error) {
    switch strings.ToLower(strings.TrimSpace(os.Getenv("MECHEXEC_BLE_IMPL"))) {
    case "sim", "simulator", "mock":
        return newSim(cfg), nil
    case "native", "goble":
        if t, err := NewNativeTransport(cfg); err == nil {
            return t, nil
        }
        // If native init fails, fall back to sim
        return newSim(cfg), nil
    default:
        if t, err := NewNativeTransport(cfg); err == nil {
            return t, nil
        }
        return newSim(cfg), nil
    }
}

func newSim(cfg *core.NetworkConfig) core.BLETransport {
    t := NewTransport()
    if cfg != nil {
        // Apply advertise interval if provided (milliseconds in config)
        if cfg.AdvertiseInterval > 0 {
            if tt, err := strconv.Atoi(strconv.Itoa(cfg.AdvertiseInterval)); err == nil {
                // cfg.AdvertiseInterval is already int; converting to duration
                d := time.Duration(tt) * time.Millisecond
                if d > 0 {
                    // direct field access is allowed inside package ble
                    t.advertiseInterval = d
                }
            }
        }
    }
    return t
}

