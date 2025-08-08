package ble

import (
    "context"
    "sync"
    "time"

    core "github.com/monster0506/mechexec/internal"
)

// Manager provides a unified interface for device discovery and connection
// management on top of a core.BLETransport implementation.
type Manager struct {
    transport core.BLETransport

    mu           sync.RWMutex
    peersByAddr  map[string]core.PeerInfo
    updates      map[int]chan core.PeerInfo
    nextSubID    int

    scanCancel   context.CancelFunc
}

// NewManager constructs a new Manager wrapping the given BLE transport.
func NewManager(transport core.BLETransport) *Manager {
    return &Manager{
        transport:  transport,
        peersByAddr: make(map[string]core.PeerInfo),
        updates:     make(map[int]chan core.PeerInfo),
    }
}

// StartDiscovery begins scanning for nearby devices and publishing updates.
// It is safe to call multiple times; subsequent calls restart scanning.
func (m *Manager) StartDiscovery(ctx context.Context) error {
    m.StopDiscovery()

    // Create an internal context so StopDiscovery can cancel regardless of caller ctx
    scanCtx, cancel := context.WithCancel(context.Background())
    m.mu.Lock()
    m.scanCancel = cancel
    m.mu.Unlock()

    advCh, err := m.transport.Scan(scanCtx)
    if err != nil {
        cancel()
        return err
    }

    go func() {
        for adv := range advCh {
            m.handleAdvertisement(adv)
        }
    }()
    return nil
}

// StopDiscovery stops the active scan, if any.
func (m *Manager) StopDiscovery() {
    m.mu.Lock()
    defer m.mu.Unlock()
    if m.scanCancel != nil {
        m.scanCancel()
        m.scanCancel = nil
    }
}

// ListPeers returns a snapshot of currently known peers.
func (m *Manager) ListPeers() []core.PeerInfo {
    m.mu.RLock()
    defer m.mu.RUnlock()
    out := make([]core.PeerInfo, 0, len(m.peersByAddr))
    for _, p := range m.peersByAddr {
        out = append(out, p)
    }
    return out
}

// GetPeer returns a peer by address if present.
func (m *Manager) GetPeer(addr string) (core.PeerInfo, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    p, ok := m.peersByAddr[addr]
    return p, ok
}

// Connect establishes a connection to a peer by address using the underlying transport.
func (m *Manager) Connect(ctx context.Context, addr string) (*core.Connection, error) {
    conn, err := m.transport.Connect(ctx, addr)
    if err != nil {
        return nil, err
    }
    // Mark as connected in the peer map
    m.mu.Lock()
    if p, ok := m.peersByAddr[addr]; ok {
        p.Connected = true
        m.peersByAddr[addr] = p
        m.publishUpdateLocked(p)
    }
    m.mu.Unlock()
    return conn, nil
}

// Subscribe returns a channel that receives peer updates (insert/update/connected changes).
// The caller should cancel the provided context to unsubscribe and close the channel.
func (m *Manager) Subscribe(ctx context.Context) <-chan core.PeerInfo {
    ch := make(chan core.PeerInfo, 16)
    m.mu.Lock()
    id := m.nextSubID
    m.nextSubID++
    m.updates[id] = ch
    m.mu.Unlock()

    go func() {
        <-ctx.Done()
        m.mu.Lock()
        delete(m.updates, id)
        close(ch)
        m.mu.Unlock()
    }()
    return ch
}

func (m *Manager) handleAdvertisement(adv *core.Advertisement) {
    now := time.Now()
    m.mu.Lock()
    p, exists := m.peersByAddr[adv.Address]
    if !exists {
        p = core.PeerInfo{
            ID:      adv.Address,
            Name:    adv.Name,
            Address: adv.Address,
            Role:    "unknown",
            OS:      "unknown",
            Arch:    "unknown",
            Tags:    map[string]string{},
        }
    }
    p.Name = adv.Name
    p.LastSeen = now
    p.SignalStrength = adv.RSSI
    m.peersByAddr[adv.Address] = p
    m.publishUpdateLocked(p)
    m.mu.Unlock()
}

func (m *Manager) publishUpdateLocked(p core.PeerInfo) {
    for _, ch := range m.updates {
        select {
        case ch <- p:
        default:
        }
    }
}

