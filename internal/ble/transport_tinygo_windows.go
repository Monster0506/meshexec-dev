//go:build windows && !goble

package ble

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	bt "tinygo.org/x/bluetooth"

	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

// tgTransport implements BLETransport using tinygo bluetooth on Windows.
// This implementation provides full BLE functionality on Windows including
// advertising simulation and GATT services using a hybrid approach.
type tgTransport struct {
	adapter *bt.Adapter
	logger  *logging.Logger

	// Hybrid simulation for unsupported features
	mu              sync.RWMutex
	localAddress    string
	localName       string
	advertisedData  []byte
	advertiseActive bool
	advertiseCancel context.CancelFunc

	// GATT service simulation
	gattServices map[string]*core.GATTService

	// Advertisement simulation for Windows peer discovery
	simAdv *Transport // Embedded simulator for advertisement
}

// NewNativeTransport returns a tinygo-backed transport on Windows.
func NewNativeTransport(cfg *core.NetworkConfig) (core.BLETransport, error) {
	return newNativeWithLogger(cfg, nil)
}

// newNativeWithLogger returns a hybrid Windows BLE transport with full functionality.
func newNativeWithLogger(cfg *core.NetworkConfig, logger *logging.Logger) (core.BLETransport, error) {
	if logger == nil {
		logger = logging.NewLogger("info")
	}

	logger.Info("Initializing Windows hybrid BLE transport", map[string]interface{}{
		"approach": "TinyGo + simulation hybrid",
	})

	// Initialize TinyGo adapter for real scanning/connecting
	ad := bt.DefaultAdapter
	if err := ad.Enable(); err != nil {
		logger.Warn("Failed to enable TinyGo BLE adapter, using full simulation", map[string]interface{}{
			"error": err.Error(),
		})
		// If TinyGo fails, fall back to full simulation
		return NewTransportWithLogger(logger), nil
	}

	// Create hybrid transport
	transport := &tgTransport{
		adapter:      ad,
		logger:       logger,
		localAddress: generateWindowsMAC(),
		localName:    getWindowsDeviceName(),
		gattServices: make(map[string]*core.GATTService),
		simAdv:       NewTransportWithLogger(logger), // Embedded simulator
	}

	logger.Info("Windows hybrid BLE transport initialized successfully", map[string]interface{}{
		"local_address":         transport.localAddress,
		"local_name":            transport.localName,
		"real_scanning":         true,
		"simulated_advertising": true,
	})

	return transport, nil
}

func (t *tgTransport) Advertise(ctx context.Context, serviceData []byte) error {
	t.logger.Info("Starting Windows hybrid BLE advertisement", map[string]interface{}{
		"service_data_length": len(serviceData),
		"method":              "simulation + local broadcast",
	})

	t.mu.Lock()
	// Stop any existing advertisement
	if t.advertiseCancel != nil {
		t.advertiseCancel()
	}

	t.advertisedData = append([]byte(nil), serviceData...)
	t.advertiseActive = true

	// Create cancellation context
	advCtx, cancel := context.WithCancel(context.Background())
	t.advertiseCancel = cancel
	t.mu.Unlock()

	// Use embedded simulator for advertisement functionality
	// This ensures Windows devices can discover each other locally
	go func() {
		defer func() {
			t.mu.Lock()
			t.advertiseActive = false
			t.advertiseCancel = nil
			t.mu.Unlock()
		}()

		// Forward to simulation layer for local mesh discovery
		if err := t.simAdv.Advertise(advCtx, serviceData); err != nil {
			t.logger.Error("Simulated advertisement failed", err, nil)
		}
	}()

	// Monitor main context for cancellation
	go func() {
		<-ctx.Done()
		t.mu.Lock()
		if t.advertiseCancel != nil {
			t.advertiseCancel()
		}
		t.mu.Unlock()
		t.logger.Info("Windows BLE advertisement stopped", nil)
	}()

	return nil
}

func (t *tgTransport) Scan(ctx context.Context) (<-chan *core.Advertisement, error) {
	t.logger.Info("Starting Windows hybrid BLE scan", map[string]interface{}{
		"method": "TinyGo + simulation combined",
	})

	out := make(chan *core.Advertisement, 32)

	go func() {
		defer close(out)
		defer t.logger.Info("Windows hybrid BLE scan stopped", nil)

		realScanCount := 0
		simScanCount := 0

		// Channel for merging real and simulated results
		realCh := make(chan *core.Advertisement, 16)
		simCh := make(chan *core.Advertisement, 16)

		// Start real TinyGo scanning in background
		go func() {
			defer close(realCh)
			scanCtx, scanCancel := context.WithCancel(ctx)
			defer scanCancel()

			_ = t.adapter.Scan(func(_ *bt.Adapter, dev bt.ScanResult) {
				realScanCount++
				adv := &core.Advertisement{
					Address:     strings.ToLower(dev.Address.String()),
					Name:        dev.LocalName(),
					ServiceData: map[string][]byte{},
					RSSI:        int(dev.RSSI),
					Timestamp:   time.Now(),
				}

				t.logger.Debug("Real BLE advertisement received", map[string]interface{}{
					"address": adv.Address,
					"name":    adv.Name,
					"rssi":    adv.RSSI,
					"source":  "TinyGo",
				})

				select {
				case realCh <- adv:
				case <-scanCtx.Done():
					return
				}
			})

			// Stop scanning when context is done
			<-scanCtx.Done()
			_ = t.adapter.StopScan()
		}()

		// Start simulated scanning for local mesh devices
		go func() {
			defer close(simCh)

			simScanOut, err := t.simAdv.Scan(ctx)
			if err != nil {
				t.logger.Error("Failed to start simulated scan", err, nil)
				return
			}

			for adv := range simScanOut {
				simScanCount++

				// Mark simulated advertisements
				t.logger.Debug("Simulated BLE advertisement received", map[string]interface{}{
					"address": adv.Address,
					"name":    adv.Name,
					"rssi":    adv.RSSI,
					"source":  "simulation",
				})

				select {
				case simCh <- adv:
				case <-ctx.Done():
					return
				}
			}
		}()

		// Merge both streams
		for {
			select {
			case adv, ok := <-realCh:
				if !ok {
					realCh = nil
				} else {
					select {
					case out <- adv:
					default:
						t.logger.Warn("Dropped real BLE advertisement - slow receiver", map[string]interface{}{
							"address": adv.Address,
						})
					}
				}
			case adv, ok := <-simCh:
				if !ok {
					simCh = nil
				} else {
					select {
					case out <- adv:
					default:
						t.logger.Warn("Dropped simulated BLE advertisement - slow receiver", map[string]interface{}{
							"address": adv.Address,
						})
					}
				}
			case <-ctx.Done():
				t.logger.Info("Windows hybrid scan completed", map[string]interface{}{
					"real_advertisements":      realScanCount,
					"simulated_advertisements": simScanCount,
					"total":                    realScanCount + simScanCount,
				})
				return
			}

			// Exit if both channels are closed
			if realCh == nil && simCh == nil {
				break
			}
		}

		t.logger.Info("Windows hybrid scan completed", map[string]interface{}{
			"real_advertisements":      realScanCount,
			"simulated_advertisements": simScanCount,
			"total":                    realScanCount + simScanCount,
		})
	}()

	return out, nil
}

func (t *tgTransport) Connect(ctx context.Context, addr string) (*core.Connection, error) {
	t.logger.Info("Attempting Windows hybrid BLE connection", map[string]interface{}{
		"target_address": addr,
	})

	// First try to determine if this is a real or simulated device
	isSimulated := t.isSimulatedDevice(addr)

	if isSimulated {
		t.logger.Debug("Connecting to simulated device", map[string]interface{}{
			"address": addr,
		})
		// Use simulation layer for local mesh connections
		return t.simAdv.Connect(ctx, addr)
	}

	// Try real TinyGo connection for external devices
	t.logger.Debug("Attempting real TinyGo connection", map[string]interface{}{
		"address": addr,
	})

	// Parse address
	var macAddr bt.MACAddress
	macAddr.Set(addr) // TinyGo Set doesn't return error

	// Validate address format manually
	if !isValidMACAddress(addr) {
		err := fmt.Errorf("invalid MAC address format: %s", addr)
		t.logger.Error("Invalid MAC address format", err, map[string]interface{}{
			"address": addr,
		})
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

	t.logger.Info("Windows hybrid BLE connection established", map[string]interface{}{
		"address":         strings.ToLower(addr),
		"mtu":             185,
		"connection_type": "real",
	})

	// MTU retrieval is characteristic-specific in tinygo API; use a reasonable default
	return &core.Connection{Address: strings.ToLower(addr), MTU: 185, Connected: true}, nil
}

func (t *tgTransport) CreateGATTService() (*core.GATTService, error) {
	t.logger.Info("Creating Windows hybrid GATT service", map[string]interface{}{
		"method": "simulation-based",
	})

	// Use simulation layer for GATT services
	service, err := t.simAdv.CreateGATTService()
	if err != nil {
		return nil, err
	}

	// Store locally for tracking
	t.mu.Lock()
	t.gattServices[service.UUID] = service
	t.mu.Unlock()

	t.logger.Info("Windows hybrid GATT service created", map[string]interface{}{
		"service_uuid":          service.UUID,
		"characteristics_count": len(service.Characteristics),
	})

	return service, nil
}

// Helper functions

// isSimulatedDevice determines if a device address belongs to our simulation layer
func (t *tgTransport) isSimulatedDevice(addr string) bool {
	// Check if it matches our local address pattern or known simulated devices
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Our own address is always simulated
	if addr == t.localAddress {
		return true
	}

	// Check if it's a locally administered MAC (indicates simulation)
	// These have the 2nd bit of the first octet set
	if len(addr) >= 2 {
		parts := strings.Split(addr, ":")
		if len(parts) > 0 {
			if firstOctet := parts[0]; len(firstOctet) >= 2 {
				// Parse first octet
				if val, err := fmt.Sscanf(firstOctet, "%x", new(int)); err == nil {
					if val != 0 {
						// Check if locally administered bit is set
						return (val & 0x02) != 0
					}
				}
			}
		}
	}

	return false
}

// generateWindowsMAC creates a locally administered MAC address for Windows
func generateWindowsMAC() string {
	b := make([]byte, 6)
	rand.Read(b)
	// Set locally administered bit (bit 1 of first octet)
	b[0] = (b[0] | 0x02) & 0xfe
	hw := net.HardwareAddr(b)
	return strings.ToUpper(hw.String())
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
