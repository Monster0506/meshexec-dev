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
)

// Transport is a minimal in-memory implementation of core.BLETransport.
//
// It simulates BLE Advertise, Scan, and Connect behaviors sufficiently for
// early integration and unit testing without requiring platform BLE drivers.
// This should be replaced or extended with a real BLE backend (e.g., go-ble/ble
// or tinygo.org/x/bluetooth) in subsequent tasks.
type Transport struct {
    mu                 sync.RWMutex
    localAddress       string
    localName          string
    advertisedData     []byte
    advertiseInterval  time.Duration

    // scanSubscribers holds channels created by Scan callers.
    scanSubscribers    map[int]chan *core.Advertisement
    nextSubscriberID   int

    // connectionState simulates connection tracking keyed by address.
    connectionState    map[string]*core.Connection

    // stopAdvertise cancels the background advertising goroutine when present.
    stopAdvertise      context.CancelFunc
}

// NewTransport creates a new simulated BLE transport.
func NewTransport() *Transport {
    name := hostnameOrDefault("meshexec")
    return &Transport{
        localAddress:      randomMAC(),
        localName:         name,
        advertiseInterval: 1 * time.Second,
        scanSubscribers:   make(map[int]chan *core.Advertisement),
        connectionState:   make(map[string]*core.Connection),
    }
}

// Advertise starts broadcasting the provided service data periodically to all
// active Scan subscribers. It runs until the context is cancelled.
func (t *Transport) Advertise(ctx context.Context, serviceData []byte) error {
    t.mu.Lock()
    // Cancel any prior advertising loop
    if t.stopAdvertise != nil {
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
        ticker := time.NewTicker(interval)
        defer ticker.Stop()
        // On first start, emit one advertisement immediately
        t.broadcastAdvertisement(addr, name, serviceData, -40)
        for {
            select {
            case <-advCtx.Done():
                return
            case <-ctx.Done():
                cancel()
                return
            case <-ticker.C:
                // Simulate variable RSSI
                rssi := -30 - rand.Intn(50)
                t.broadcastAdvertisement(addr, name, serviceData, rssi)
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
    t.mu.Unlock()

    go func() {
        <-ctx.Done()
        t.mu.Lock()
        delete(t.scanSubscribers, id)
        close(ch)
        t.mu.Unlock()
    }()

    return ch, nil
}

// Connect simulates establishing a connection to a device by address.
// For now, only the local device address is recognized.
func (t *Transport) Connect(ctx context.Context, addr string) (*core.Connection, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    t.mu.Lock()
    defer t.mu.Unlock()

    if !isMAC(addr) {
        return nil, errors.New("invalid address format; expected MAC-like string XX:XX:XX:XX:XX:XX")
    }

    conn, exists := t.connectionState[addr]
    if !exists {
        if addr != t.localAddress {
            return nil, errors.New("simulated transport: remote address not found")
        }
        conn = &core.Connection{Address: addr, MTU: 185, Connected: true}
        t.connectionState[addr] = conn
    } else {
        conn.Connected = true
    }
    return conn, nil
}

// CreateGATTService returns a minimal placeholder GATT service description.
func (t *Transport) CreateGATTService() (*core.GATTService, error) {
    // Placeholder UUID for MechExec; to be replaced with configured UUID.
    return &core.GATTService{
        UUID: "0000-MECH-EXEC-0000",
        Characteristics: []core.GATTCharacteristic{
            {UUID: "0000-MECH-CHAR-0001"},
        },
    }, nil
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
    for _, sub := range t.scanSubscribers {
        // Non-blocking send; drop if subscriber is slow
        select {
        case sub <- adv:
        default:
        }
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

