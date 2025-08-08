package ble

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

// Transport is a minimal in-memory implementation of core.BLETransport.
//
// It simulates BLE Advertise, Scan, and Connect behaviors sufficiently for
// early integration and unit testing without requiring platform BLE drivers.
// This should be replaced or extended with a real BLE backend (e.g., go-ble/ble
// or tinygo.org/x/bluetooth) in subsequent tasks.
type Transport struct {
	mu                sync.RWMutex
	logger            *logging.Logger
	localAddress      string
	localName         string
	advertisedData    []byte
	advertiseInterval time.Duration

	// scanSubscribers holds channels created by Scan callers.
	scanSubscribers  map[int]chan *core.Advertisement
	nextSubscriberID int

	// connectionState simulates connection tracking keyed by address.
	connectionState map[string]*core.Connection

	// stopAdvertise cancels the background advertising goroutine when present.
	stopAdvertise context.CancelFunc
}

// NewTransport creates a new simulated BLE transport.
func NewTransport() *Transport {
	return NewTransportWithLogger(nil)
}

// NewTransportWithLogger creates a new simulated BLE transport with a logger.
func NewTransportWithLogger(logger *logging.Logger) *Transport {
	if logger == nil {
		logger = logging.NewLogger("info")
	}

	name := hostnameOrDefault("meshexec")
	address := randomMAC()

	transport := &Transport{
		logger:            logger,
		localAddress:      address,
		localName:         name,
		advertiseInterval: 1 * time.Second,
		scanSubscribers:   make(map[int]chan *core.Advertisement),
		connectionState:   make(map[string]*core.Connection),
	}

	logger.Info("Simulated BLE transport initialized", map[string]interface{}{
		"local_address":         address,
		"local_name":            name,
		"advertise_interval_ms": transport.advertiseInterval.Milliseconds(),
	})

	return transport
}

// Advertise starts broadcasting the provided service data periodically to all
// active Scan subscribers. It runs until the context is cancelled.
func (t *Transport) Advertise(ctx context.Context, serviceData []byte) error {
	t.logger.Info("Starting BLE advertisement", map[string]interface{}{
		"service_data_length": len(serviceData),
		"interval_ms":         t.advertiseInterval.Milliseconds(),
	})

	t.mu.Lock()
	// Cancel any prior advertising loop
	if t.stopAdvertise != nil {
		t.logger.Debug("Stopping previous advertisement", nil)
		t.stopAdvertise()
		t.stopAdvertise = nil
	}
	t.advertisedData = append([]byte(nil), serviceData...)
	// Create a child context we can cancel independently when a new call arrives
	advCtx, cancel := context.WithCancel(context.Background())
	t.stopAdvertise = cancel
	interval := t.advertiseInterval
	addr := t.localAddress
	name := t.localName
	t.mu.Unlock()

	go func() {
		t.logger.Debug("Advertisement broadcast loop started", map[string]interface{}{
			"address": addr,
			"name":    name,
		})

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer t.logger.Debug("Advertisement broadcast loop ended", nil)

		broadcastCount := 0
		// On first start, emit one advertisement immediately
		t.broadcastAdvertisement(addr, name, serviceData, -40)
		broadcastCount++

		for {
			select {
			case <-advCtx.Done():
				t.logger.Debug("Advertisement stopped via context", map[string]interface{}{
					"total_broadcasts": broadcastCount,
				})
				return
			case <-ctx.Done():
				cancel()
				t.logger.Info("Advertisement stopped - context cancelled", map[string]interface{}{
					"total_broadcasts": broadcastCount,
				})
				return
			case <-ticker.C:
				// Simulate variable RSSI
				rssi := -30 - rand.Intn(50)
				t.broadcastAdvertisement(addr, name, serviceData, rssi)
				broadcastCount++

				if broadcastCount%10 == 0 {
					t.logger.Debug("Advertisement broadcast progress", map[string]interface{}{
						"total_broadcasts": broadcastCount,
						"current_rssi":     rssi,
					})
				}
			}
		}
	}()

	return nil
}

// Scan subscribes the caller to receive discovered advertisements. In this
// simulated implementation, it receives locally advertised frames and can be
// extended later to relay real discoveries from a hardware backend.
func (t *Transport) Scan(ctx context.Context) (<-chan *core.Advertisement, error) {
	ch := make(chan *core.Advertisement, 8)

	t.mu.Lock()
	id := t.nextSubscriberID
	t.nextSubscriberID++
	t.scanSubscribers[id] = ch
	t.logger.Debug("New scan subscription created", map[string]interface{}{
		"subscription_id":   id,
		"total_subscribers": len(t.scanSubscribers),
	})
	t.mu.Unlock()

	go func() {
		<-ctx.Done()
		t.mu.Lock()
		delete(t.scanSubscribers, id)
		close(ch)
		t.logger.Debug("Scan subscription closed", map[string]interface{}{
			"subscription_id":       id,
			"remaining_subscribers": len(t.scanSubscribers),
		})
		t.mu.Unlock()
	}()

	t.logger.Info("BLE scan started", map[string]interface{}{
		"subscription_id": id,
	})
	return ch, nil
}

// Connect simulates establishing a connection to a device by address.
// For now, only the local device address is recognized.
func (t *Transport) Connect(ctx context.Context, addr string) (*core.Connection, error) {
	t.logger.Info("Attempting to connect to device", map[string]interface{}{
		"target_address": addr,
		"local_address":  t.localAddress,
	})

	select {
	case <-ctx.Done():
		t.logger.Warn("Connection attempt cancelled by context", map[string]interface{}{
			"target_address": addr,
		})
		return nil, ctx.Err()
	default:
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if !isMAC(addr) {
		t.logger.Error("Invalid MAC address format", nil, map[string]interface{}{
			"address": addr,
		})
		return nil, errors.New("invalid address format; expected MAC-like string XX:XX:XX:XX:XX:XX")
	}

	conn, exists := t.connectionState[addr]
	if !exists {
		if addr != t.localAddress {
			t.logger.Warn("Connection attempt to unknown remote address", map[string]interface{}{
				"target_address": addr,
				"local_address":  t.localAddress,
			})
			return nil, errors.New("simulated transport: remote address not found")
		}
		conn = &core.Connection{Address: addr, MTU: 185, Connected: true}
		t.connectionState[addr] = conn
		t.logger.Info("New connection established", map[string]interface{}{
			"address": addr,
			"mtu":     conn.MTU,
		})
	} else {
		conn.Connected = true
		t.logger.Debug("Existing connection reactivated", map[string]interface{}{
			"address": addr,
			"mtu":     conn.MTU,
		})
	}
	return conn, nil
}

// CreateGATTService returns a minimal placeholder GATT service description.
func (t *Transport) CreateGATTService() (*core.GATTService, error) {
	service := &core.GATTService{
		UUID: "0000-MECH-EXEC-0000",
		Characteristics: []core.GATTCharacteristic{
			{UUID: "0000-MECH-CHAR-0001"},
		},
	}

	t.logger.Info("Created GATT service", map[string]interface{}{
		"service_uuid":          service.UUID,
		"characteristics_count": len(service.Characteristics),
	})

	return service, nil
}

// broadcastAdvertisement sends an advertisement to all active subscribers.
func (t *Transport) broadcastAdvertisement(addr, name string, serviceData []byte, rssi int) {
	adv := &core.Advertisement{
		Address: addr,
		Name:    name,
		ServiceData: map[string][]byte{
			"meshexec": append([]byte(nil), serviceData...),
		},
		RSSI:      rssi,
		Timestamp: time.Now(),
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	sent := 0
	dropped := 0
	for _, sub := range t.scanSubscribers {
		// Non-blocking send; drop if subscriber is slow
		select {
		case sub <- adv:
			sent++
		default:
			dropped++
		}
	}

	// Log broadcast statistics periodically
	if dropped > 0 {
		t.logger.Warn("Advertisement broadcast dropped to slow subscribers", map[string]interface{}{
			"sent":    sent,
			"dropped": dropped,
			"rssi":    rssi,
		})
	}
}

func hostnameOrDefault(def string) string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return def
}

func randomMAC() string {
	// Use a locally administered MAC prefix (x2)
	b := make([]byte, 6)
	rand.Seed(time.Now().UnixNano())
	rand.Read(b)
	b[0] = (b[0] | 0x02) & 0xfe
	hw := net.HardwareAddr(b)
	return hw.String()
}

func isMAC(s string) bool {
	parts := strings.Split(s, ":")
	if len(parts) != 6 {
		return false
	}
	// A minimal sanity check; we rely on format only here
	return true
}
