package mesh

import (
	"context"
	"errors"
	"sync"

	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/ble"
	"github.com/monster0506/meshexec/internal/logging"
)

// Node implements core.MeshNode, orchestrating BLE discovery/advertising
// and providing a simple pub-sub for mesh messages (transport wiring TBD).
type Node struct {
	transport core.BLETransport
	manager   *ble.Manager
	cfg       *core.NetworkConfig
	localPeer core.PeerInfo

	advCancel context.CancelFunc

	subsMu sync.RWMutex
	subs   map[core.MessageType][]chan *core.MeshMessage

	startedMu sync.Mutex
	started   bool
}

// NewNode constructs a MeshNode from a BLE transport and network config.
func NewNode(transport core.BLETransport, cfg *core.NetworkConfig, local core.PeerInfo) *Node {
	logger := logging.NewLogger("info") // Default logger for mesh node
	return &Node{
		transport: transport,
		manager:   ble.NewManager(transport, logger),
		cfg:       cfg,
		localPeer: local,
		subs:      make(map[core.MessageType][]chan *core.MeshMessage),
	}
}

// NewNodeFromConfig builds a native/sim transport using the BLE factory and returns a node.
func NewNodeFromConfig(cfg *core.Config) (*Node, error) {
	logger := logging.NewLogger("info") // Create logger for the factory
	t, err := ble.NewWithLogger(&cfg.Network, logger)
	if err != nil {
		return nil, err
	}
	local := core.PeerInfo{
		ID:      cfg.Device.Name,
		Name:    cfg.Device.Name,
		Address: "", // filled by transport-specific logic when available
		Role:    cfg.Device.Role,
		OS:      cfg.Device.OS,
		Arch:    cfg.Device.Arch,
		Tags:    cfg.Device.Tags,
	}
	return NewNode(t, &cfg.Network, local), nil
}

// Start begins BLE advertising and discovery.
func (n *Node) Start(ctx context.Context) error {
	n.startedMu.Lock()
	if n.started {
		n.startedMu.Unlock()
		return nil
	}
	n.started = true
	n.startedMu.Unlock()

	// Try to create GATT service; if unsupported on this platform/transport, continue without it
	if _, err := n.transport.CreateGATTService(); err != nil {
		// proceed without GATT service (e.g., Windows central-only path)
	}

	// Start advertising in a cancellable context
	advCtx, advCancel := context.WithCancel(context.Background())
	n.advCancel = advCancel
	// Use a minimal serviceData marker; future: encode discovery metadata
	if err := n.transport.Advertise(advCtx, []byte("meshexec")); err != nil {
		// proceed with scanning only when advertising is unavailable
		advCancel()
	}

	// Start discovery
	if err := n.manager.StartDiscovery(ctx); err != nil {
		n.Stop()
		return err
	}

	return nil
}

// Stop halts advertising and discovery.
func (n *Node) Stop() error {
	n.startedMu.Lock()
	if !n.started {
		n.startedMu.Unlock()
		return nil
	}
	n.started = false
	n.startedMu.Unlock()

	if n.advCancel != nil {
		n.advCancel()
		n.advCancel = nil
	}
	n.manager.StopDiscovery()
	return nil
}

// SendMessage publishes a message into the mesh.
// NOTE: BLE message transmission is implemented in task 6.2.
func (n *Node) SendMessage(msg *core.MeshMessage) error {
	// For now, accept but not transmit. This will be wired to BLE GATT in 6.2.
	if msg == nil {
		return errors.New("nil message")
	}
	// Locally publish to subscribers of this type.
	n.subsMu.RLock()
	subs := append([]chan *core.MeshMessage(nil), n.subs[msg.Type]...)
	n.subsMu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
		}
	}
	return nil
}

// Subscribe returns a channel for a given message type.
func (n *Node) Subscribe(msgType core.MessageType) <-chan *core.MeshMessage {
	ch := make(chan *core.MeshMessage, 16)
	n.subsMu.Lock()
	n.subs[msgType] = append(n.subs[msgType], ch)
	n.subsMu.Unlock()
	return ch
}

// GetPeers returns the current discovered peers.
func (n *Node) GetPeers() []core.PeerInfo {
	return n.manager.ListPeers()
}
