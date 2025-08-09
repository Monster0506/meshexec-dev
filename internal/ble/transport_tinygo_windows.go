//go:build windows && !goble

package ble

import (
    "context"
    "errors"
    "fmt"
    "os"
    "strings"
    "sync"
    "time"

    bt "tinygo.org/x/bluetooth"

    core "github.com/monster0506/meshexec/internal"
    "github.com/monster0506/meshexec/internal/logging"
)

// tgTransport implements BLETransport using tinygo bluetooth on Windows.
type tgTransport struct {
	adapter *bt.Adapter
	logger  *logging.Logger

	mu              sync.RWMutex
	localAddress    string
	localName       string
    advertisedData  []byte
    advertiseActive bool
    advertiseCancel context.CancelFunc

    gattServices map[string]*core.GATTService
    // Note: Windows peripheral/server not available via tinygo.org/x/bluetooth in this toolchain
}

// NewNativeTransport returns a tinygo-backed transport on Windows.
func NewNativeTransport(cfg *core.NetworkConfig) (core.BLETransport, error) {
	return newNativeWithLogger(cfg, nil)
}

func newNativeWithLogger(cfg *core.NetworkConfig, logger *logging.Logger) (core.BLETransport, error) {
	if logger == nil {
		logger = logging.NewLogger("info")
	}

    logger.Info("Initializing Windows TinyGo BLE transport", map[string]interface{}{
        "approach": "TinyGo WinRT",
    })

	ad := bt.DefaultAdapter
	if err := ad.Enable(); err != nil {
        logger.Error("Failed to enable TinyGo BLE adapter", err, nil)
        return nil, err
	}

    // Create transport
    transport := &tgTransport{
		adapter:      ad,
		logger:       logger,
        localAddress: "",
		localName:    getWindowsDeviceName(),
		gattServices: make(map[string]*core.GATTService),
	}

    if mac, err := ad.Address(); err == nil {
        transport.localAddress = strings.ToLower(mac.String())
    }

    logger.Info("Windows TinyGo BLE transport initialized", map[string]interface{}{
		"local_address":         transport.localAddress,
		"local_name":            transport.localName,
        "real_scanning":         true,
	})

	return transport, nil
}

func (t *tgTransport) Advertise(ctx context.Context, serviceData []byte) error {
    // Advertising (peripheral role) is not supported in this environment via tinygo.org/x/bluetooth.
    t.logger.Error("Advertising not supported on Windows TinyGo backend (no simulation)", errors.New("unsupported"), nil)
    return fmt.Errorf("windows advertising not supported by tinygo backend in this toolchain")
}

func (t *tgTransport) Scan(ctx context.Context) (<-chan *core.Advertisement, error) {
    t.logger.Info("Starting Windows TinyGo BLE scan", nil)

	out := make(chan *core.Advertisement, 32)

	go func() {
		defer close(out)
        defer t.logger.Info("Windows TinyGo BLE scan stopped", nil)

        scanCtx, scanCancel := context.WithCancel(ctx)
        defer scanCancel()

        _ = t.adapter.Scan(func(_ *bt.Adapter, dev bt.ScanResult) {
            adv := &core.Advertisement{
                Address:     strings.ToLower(dev.Address.String()),
                Name:        dev.LocalName(),
                ServiceData: map[string][]byte{},
                RSSI:        int(dev.RSSI),
                Timestamp:   time.Now(),
            }
            // TODO: when TinyGo exposes service data on Windows, populate here.
            select {
            case out <- adv:
            default:
                // drop if receiver slow
            }
        })

        <-scanCtx.Done()
        _ = t.adapter.StopScan()
	}()

	return out, nil
}

func (t *tgTransport) Connect(ctx context.Context, addr string) (*core.Connection, error) {
    t.logger.Info("Attempting Windows TinyGo BLE connection", map[string]interface{}{
		"target_address": addr,
	})

    // Parse address
    var macAddr bt.MACAddress
    macAddr.Set(addr)

    if !isValidMACAddress(addr) {
        err := fmt.Errorf("invalid MAC address format: %s", addr)
        t.logger.Error("Invalid MAC address format", err, map[string]interface{}{"address": addr})
        return nil, err
    }

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

    t.logger.Info("Windows TinyGo BLE connection established", map[string]interface{}{
		"address":         strings.ToLower(addr),
		"mtu":             185,
	})

	// MTU retrieval is characteristic-specific in tinygo API; use a reasonable default
	return &core.Connection{Address: strings.ToLower(addr), MTU: 185, Connected: true}, nil
}

func (t *tgTransport) CreateGATTService() (*core.GATTService, error) {
    // GATT server not supported via tinygo.org/x/bluetooth in this toolchain.
    t.logger.Error("GATT service not supported on Windows TinyGo backend (no simulation)", errors.New("unsupported"), nil)
    return nil, fmt.Errorf("windows gatt server not supported by tinygo backend in this toolchain")
}

// getWindowsDeviceName gets the Windows computer name
func getWindowsDeviceName() string {
	if name := strings.TrimSpace(os.Getenv("COMPUTERNAME")); name != "" {
		return name
	}
	if name := strings.TrimSpace(os.Getenv("HOSTNAME")); name != "" {
		return name
	}
	return "Windows-MeshExec"
}

// isValidMACAddress validates a MAC address format
func isValidMACAddress(addr string) bool {
	parts := strings.Split(addr, ":")
	if len(parts) != 6 {
		return false
	}
	for _, part := range parts {
		if len(part) != 2 {
			return false
		}
		for _, char := range part {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
				return false
			}
		}
	}
	return true
}
