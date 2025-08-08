package ble

import (
    "context"
    "fmt"
    "strings"
    "sync"
    "time"

    core "github.com/monster0506/meshexec/internal"
    "github.com/monster0506/meshexec/internal/logging"
)

// Manager provides a unified interface for device discovery and connection
// management on top of a core.BLETransport implementation.
type Manager struct {
    transport core.BLETransport
    logger    *logging.Logger

    mu           sync.RWMutex
    peersByAddr  map[string]core.PeerInfo
    updates      map[int]chan core.PeerInfo
    nextSubID    int

    scanCancel   context.CancelFunc
}

// NewManager constructs a new Manager wrapping the given BLE transport.
func NewManager(transport core.BLETransport, logger *logging.Logger) *Manager {
    if logger == nil {
        logger = logging.NewLogger("info")
    }
    
    manager := &Manager{
        transport:   transport,
        logger:      logger,
        peersByAddr: make(map[string]core.PeerInfo),
        updates:     make(map[int]chan core.PeerInfo),
    }
    
    logger.Info("BLE Manager initialized", map[string]interface{}{
        "transport_type": getTransportType(transport),
    })
    
    return manager
}

// getTransportType returns a string representation of the transport type
func getTransportType(transport core.BLETransport) string {
    // Check type by examining the type name to avoid import cycles
    typeName := fmt.Sprintf("%T", transport)
    
    // Check for simulated transport patterns
    if strings.Contains(typeName, "*ble.Transport") || strings.Contains(typeName, "Transport") {
        return "simulated"
    }
    
    // Check for native transport patterns
    if strings.Contains(typeName, "native") {
        return "native"
    }
    
    // Default fallback
    return "unknown"
}

// StartDiscovery begins scanning for nearby devices and publishing updates.
// It is safe to call multiple times; subsequent calls restart scanning.
func (m *Manager) StartDiscovery(ctx context.Context) error {
    m.logger.Info("Starting BLE discovery", nil)
    m.StopDiscovery()

    // Create an internal context so StopDiscovery can cancel regardless of caller ctx
    scanCtx, cancel := context.WithCancel(context.Background())
    m.mu.Lock()
    m.scanCancel = cancel
    m.mu.Unlock()

    m.logger.Debug("Initiating BLE scan", nil)
    advCh, err := m.transport.Scan(scanCtx)
    if err != nil {
        cancel()
        m.logger.Error("Failed to start BLE scan", err, nil)
        return err
    }

    m.logger.Info("BLE scan started successfully", nil)
    go func() {
        m.logger.Debug("Advertisement processing goroutine started", nil)
        advCount := 0
        for adv := range advCh {
            advCount++
            m.logger.Debug("Received advertisement", map[string]interface{}{
                "address": adv.Address,
                "name": adv.Name,
                "rssi": adv.RSSI,
                "count": advCount,
            })
            m.handleAdvertisement(adv)
        }
        m.logger.Debug("Advertisement processing goroutine ended", map[string]interface{}{
            "total_advertisements": advCount,
        })
    }()
    return nil
}

// StopDiscovery stops the active scan, if any.
func (m *Manager) StopDiscovery() {
    m.mu.Lock()
    defer m.mu.Unlock()
    if m.scanCancel != nil {
        m.logger.Info("Stopping BLE discovery", nil)
        m.scanCancel()
        m.scanCancel = nil
        m.logger.Debug("BLE discovery stopped", nil)
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
    m.logger.Debug("Listed peers", map[string]interface{}{
        "peer_count": len(out),
    })
    return out
}

// GetPeer returns a peer by address if present.
func (m *Manager) GetPeer(addr string) (core.PeerInfo, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    p, ok := m.peersByAddr[addr]
    m.logger.Debug("Get peer by address", map[string]interface{}{
        "address": addr,
        "found": ok,
        "name": func() string {
            if ok {
                return p.Name
            }
            return ""
        }(),
    })
    return p, ok
}

// Connect establishes a connection to a peer by address using the underlying transport.
func (m *Manager) Connect(ctx context.Context, addr string) (*core.Connection, error) {
    m.logger.Info("Attempting to connect to peer", map[string]interface{}{
        "address": addr,
    })
    
    conn, err := m.transport.Connect(ctx, addr)
    if err != nil {
        m.logger.Error("Failed to connect to peer", err, map[string]interface{}{
            "address": addr,
        })
        return nil, err
    }
    
    m.logger.Info("Successfully connected to peer", map[string]interface{}{
        "address": addr,
        "mtu": conn.MTU,
    })
    
    // Mark as connected in the peer map
    m.mu.Lock()
    if p, ok := m.peersByAddr[addr]; ok {
        p.Connected = true
        m.peersByAddr[addr] = p
        m.publishUpdateLocked(p)
        m.logger.Debug("Updated peer connection status", map[string]interface{}{
            "address": addr,
            "name": p.Name,
            "connected": true,
        })
    } else {
        m.logger.Warn("Connected to unknown peer - not in peer list", map[string]interface{}{
            "address": addr,
        })
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
    m.logger.Debug("New peer update subscription", map[string]interface{}{
        "subscription_id": id,
        "total_subscriptions": len(m.updates),
    })
    m.mu.Unlock()

    go func() {
        <-ctx.Done()
        m.mu.Lock()
        delete(m.updates, id)
        close(ch)
        m.logger.Debug("Peer update subscription closed", map[string]interface{}{
            "subscription_id": id,
            "remaining_subscriptions": len(m.updates),
        })
        m.mu.Unlock()
    }()
    return ch
}

func (m *Manager) handleAdvertisement(adv *core.Advertisement) {
    now := time.Now()
    m.mu.Lock()
    p, exists := m.peersByAddr[adv.Address]
    
    if !exists {
        m.logger.Info("Discovered new peer", map[string]interface{}{
            "address": adv.Address,
            "name": adv.Name,
            "rssi": adv.RSSI,
        })
        p = core.PeerInfo{
            ID:      adv.Address,
            Name:    adv.Name,
            Address: adv.Address,
            Role:    "unknown",
            OS:      "unknown",
            Arch:    "unknown",
            Tags:    map[string]string{},
        }
    } else {
        // Log significant RSSI changes (more than 10 dBm difference)
        rssiDiff := adv.RSSI - p.SignalStrength
        if rssiDiff > 10 || rssiDiff < -10 {
            m.logger.Debug("Significant RSSI change for peer", map[string]interface{}{
                "address": adv.Address,
                "name": adv.Name,
                "old_rssi": p.SignalStrength,
                "new_rssi": adv.RSSI,
                "change": rssiDiff,
            })
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
    sent := 0
    dropped := 0
    for _, ch := range m.updates {
        select {
        case ch <- p:
            sent++
        default:
            dropped++
        }
    }
    
    if dropped > 0 {
        m.logger.Warn("Dropped peer updates due to slow subscribers", map[string]interface{}{
            "peer_address": p.Address,
            "sent": sent,
            "dropped": dropped,
            "total_subscribers": len(m.updates),
        })
    } else if len(m.updates) > 0 {
        m.logger.Debug("Published peer update", map[string]interface{}{
            "peer_address": p.Address,
            "peer_name": p.Name,
            "subscribers": sent,
        })
    }
}

