package mesh

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"sync"
	"time"

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

	// message subscriptions
	subsMu sync.RWMutex
	subs   map[core.MessageType][]chan *core.MeshMessage

	// rx cancellation for BLE notifications
	rxCancel context.CancelFunc

	// reassembly state for fragmented frames
	reasmMu sync.Mutex
	reasm   map[string]*reassembler

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
		reasm:     make(map[string]*reassembler),
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
		_ = err
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
		_ = n.Stop()
		return err
	}

	// Start BLE notification receiver if transport supports it
	if sub, ok := n.transport.(interface {
		SubscribeWriteNotifications(ctx context.Context) (<-chan []byte, func(), error)
	}); ok {
		rxCtx, rxCancel := context.WithCancel(context.Background())
		n.rxCancel = rxCancel
		go func() {
			backoff := time.Millisecond * 200
			for {
				ch, unsub, err := sub.SubscribeWriteNotifications(rxCtx)
				if err != nil {
					time.Sleep(backoff)
					if backoff < 5*time.Second {
						backoff *= 2
					}
					continue
				}
				backoff = 200 * time.Millisecond
				for {
					select {
					case <-rxCtx.Done():
						unsub()
						return
					case b, ok := <-ch:
						if !ok {
							// subscribe loop ended; resubscribe
							unsub()
							goto RESUB
						}
						if full := n.tryReassemble(b); full != nil {
							var m core.MeshMessage
							if err := json.Unmarshal(full, &m); err == nil {
								// Preserve raw JSON payload so subscribers can deserialize full message (e.g., results)
								m.Payload = append([]byte(nil), full...)
								n.publishLocal(&m)
							}
						}
					}
				}
			RESUB:
				// loop to resubscribe
				continue
			}
		}()
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
	if n.rxCancel != nil {
		n.rxCancel()
		n.rxCancel = nil
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

	// Attempt BLE send via transport if supported
	b, err := json.Marshal(msg)
	if err == nil {
		mtu := 185
		if m, ok := n.transport.(interface{ EffectiveMTU() int }); ok {
			if v := m.EffectiveMTU(); v > 0 {
				mtu = v
			}
		}
		frames := n.buildFramesWithMTU(msg.ID, b, mtu)

		// Use notifications when available (peripheral role)
		if sender, ok := n.transport.(interface {
			SendNotification(ctx context.Context, data []byte) error
		}); ok {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			for _, fr := range frames {
				_ = sender.SendNotification(ctx, fr)
			}
			cancel()
		}

		// Use central broadcast when available (central role) to push frames to peers
		if broadcaster, ok := n.transport.(interface {
			CentralBroadcast(ctx context.Context, data []byte) error
		}); ok {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			for _, fr := range frames {
				_ = broadcaster.CentralBroadcast(ctx, fr)
			}
			cancel()
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

func (n *Node) publishLocal(msg *core.MeshMessage) {
	n.subsMu.RLock()
	subs := append([]chan *core.MeshMessage(nil), n.subs[msg.Type]...)
	n.subsMu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
		}
	}
}

// --- Fragmentation/Reassembly helpers ---

const (
	frameMagic0     = byte('M')
	frameMagic1     = byte('X')
	frameVersion    = byte(1)
	frameHeaderLen  = 1 + 1 + 1 + 1 + 16 // M X ver total seq md5(16)
	maxFramePayload = 160
)

func (n *Node) buildFramesWithMTU(messageID string, data []byte, mtu int) [][]byte {
	payload := maxFramePayload
	if mtu > 30 { // rough safety margin for ATT/GATT headers
		p := mtu - 25
		if p < payload {
			payload = p
		}
		if payload < 64 {
			payload = 64
		}
	}
	total := (len(data) + payload - 1) / payload
	if total <= 1 {
		return [][]byte{data}
	}
	if total > 255 {
		// truncate to 255 frames maximum
		total = 255
	}
	sum := md5.Sum([]byte(messageID))
	frames := make([][]byte, 0, total)
	offset := 0
	for seq := 0; seq < total; seq++ {
		remaining := len(data) - offset
		take := payload
		if remaining < take {
			take = remaining
		}
		payload := data[offset : offset+take]
		offset += take
		fr := make([]byte, 0, frameHeaderLen+len(payload))
		fr = append(fr, frameMagic0, frameMagic1, frameVersion, byte(total), byte(seq))
		fr = append(fr, sum[:]...)
		fr = append(fr, payload...)
		frames = append(frames, fr)
		if offset >= len(data) {
			break
		}
	}
	return frames
}

type reassembler struct {
	total   int
	seen    int
	chunks  map[int][]byte
	created time.Time
}

// tryReassemble either passes through raw JSON, or buffers fragment frames until complete and returns full payload.
func (n *Node) tryReassemble(b []byte) []byte {
	if len(b) >= 2 && b[0] == frameMagic0 && b[1] == frameMagic1 && len(b) >= (frameHeaderLen) {
		if b[2] != frameVersion {
			return nil
		}
		total := int(b[3])
		seq := int(b[4])
		key := string(b[5 : 5+16])
		payload := b[5+16:]
		n.reasmMu.Lock()
		ra := n.reasm[key]
		if ra == nil {
			ra = &reassembler{total: total, chunks: make(map[int][]byte), created: time.Now()}
			n.reasm[key] = ra
		}
		if _, exists := ra.chunks[seq]; !exists {
			ra.chunks[seq] = append([]byte(nil), payload...)
			ra.seen++
		}
		done := ra.seen >= ra.total
		if done {
			// assemble in order
			var full []byte
			for i := 0; i < ra.total; i++ {
				if part, ok := ra.chunks[i]; ok {
					full = append(full, part...)
				} else {
					// missing chunk; keep state
					n.reasmMu.Unlock()
					return nil
				}
			}
			delete(n.reasm, key)
			n.reasmMu.Unlock()
			return full
		}
		n.reasmMu.Unlock()
		return nil
	}
	// Assume raw JSON
	return b
}
