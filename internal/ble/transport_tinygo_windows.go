//go:build windows && !goble

package ble

import (
    "context"
    "errors"
    "strings"
    "time"

    bt "tinygo.org/x/bluetooth"

    core "github.com/monster0506/meshexec/internal"
    "github.com/monster0506/meshexec/internal/logging"
)

// tgTransport implements BLETransport using tinygo bluetooth on Windows.
// Windows currently supports central mode (scan/connect) but not peripheral
// advertisement/services via this library. Advertise and CreateGATTService
// will return an error.
type tgTransport struct {
    adapter *bt.Adapter
    logger  *logging.Logger
}

// NewNativeTransport returns a tinygo-backed transport on Windows.
func NewNativeTransport(cfg *core.NetworkConfig) (core.BLETransport, error) {
    return newNativeWithLogger(cfg, nil)
}

// newNativeWithLogger returns a tinygo-backed transport on Windows with logging.
func newNativeWithLogger(cfg *core.NetworkConfig, logger *logging.Logger) (core.BLETransport, error) {
    if logger == nil {
        logger = logging.NewLogger("info")
    }
    
    logger.Info("Initializing TinyGo BLE transport on Windows", nil)
    
    ad := bt.DefaultAdapter
    if err := ad.Enable(); err != nil {
        logger.Error("Failed to enable BLE adapter", err, nil)
        return nil, err
    }
    
    logger.Info("TinyGo BLE transport initialized successfully", nil)
    return &tgTransport{adapter: ad, logger: logger}, nil
}

func (t *tgTransport) Advertise(ctx context.Context, serviceData []byte) error {
    t.logger.Warn("Advertisement not supported on Windows TinyGo backend", map[string]interface{}{
        "service_data_length": len(serviceData),
    })
    return errors.New("advertising not supported on Windows backend (tinygo bluetooth)")
}

func (t *tgTransport) Scan(ctx context.Context) (<-chan *core.Advertisement, error) {
    t.logger.Info("Starting TinyGo BLE scan on Windows", nil)
    
    out := make(chan *core.Advertisement, 32)
    go func() {
        defer close(out)
        defer t.logger.Info("TinyGo BLE scan stopped", nil)
        
        scanCount := 0
        _ = t.adapter.Scan(func(_ *bt.Adapter, dev bt.ScanResult) {
            scanCount++
            adv := &core.Advertisement{
                Address:     strings.ToLower(dev.Address.String()),
                Name:        dev.LocalName(),
                ServiceData: map[string][]byte{},
                RSSI:        int(dev.RSSI),
                Timestamp:   time.Now(),
            }
            
            t.logger.Debug("TinyGo BLE advertisement received", map[string]interface{}{
                "address": adv.Address,
                "name": adv.Name,
                "rssi": adv.RSSI,
                "scan_count": scanCount,
            })
            
            select { 
            case out <- adv: 
            default:
                t.logger.Warn("Dropped TinyGo BLE advertisement - slow receiver", map[string]interface{}{
                    "address": adv.Address,
                })
            }
        })
        <-ctx.Done()
        _ = t.adapter.StopScan()
        
        t.logger.Info("TinyGo BLE scan completed", map[string]interface{}{
            "total_advertisements": scanCount,
        })
    }()
    return out, nil
}

func (t *tgTransport) Connect(ctx context.Context, addr string) (*core.Connection, error) {
    t.logger.Info("Attempting TinyGo BLE connection", map[string]interface{}{
        "target_address": addr,
    })
    
    // Parse address
    var macAddr bt.MACAddress
    macAddr.Set(addr)
    dev, err := t.adapter.Connect(bt.Address{MACAddress: macAddr}, bt.ConnectionParams{})
    if err != nil {
        t.logger.Error("TinyGo BLE connection failed", err, map[string]interface{}{
            "target_address": addr,
        })
        return nil, err
    }
    
    // Best-effort timeout context to ensure we honor ctx cancellation
    go func() {
        select {
        case <-ctx.Done():
            t.logger.Debug("TinyGo BLE connection cancelled, disconnecting", map[string]interface{}{
                "address": addr,
            })
            _ = dev.Disconnect()
        }
    }()
    
    t.logger.Info("TinyGo BLE connection established", map[string]interface{}{
        "address": strings.ToLower(addr),
        "mtu": 185,
    })
    
    // MTU retrieval is characteristic-specific in tinygo API; use a reasonable default
    return &core.Connection{Address: strings.ToLower(addr), MTU: 185, Connected: true}, nil
}

func (t *tgTransport) CreateGATTService() (*core.GATTService, error) {
    t.logger.Warn("GATT service creation not supported on Windows TinyGo backend", nil)
    return nil, errors.New("GATT server not supported on Windows backend (tinygo bluetooth)")
}

