package mesh

import (
	"context"
	"crypto/md5"
	"errors"
	"sync"
	"time"

	core "github.com/monster0506/meshexec/internal"
)

// Node implements core.MeshNode, orchestrating BLE discovery/advertising
// and providing a simple pub-sub for mesh messages (transport wiring TBD).
type Node struct {
	transport interface{}
	manager   interface{}
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
func NewNode(transport interface{}, cfg *core.NetworkConfig, local core.PeerInfo) *Node {
	return &Node{
		transport: transport,
		manager:   nil,
		cfg:       cfg,
		localPeer: local,
		subs:      make(map[core.MessageType][]chan *core.MeshMessage),
		reasm:     make(map[string]*reassembler),
	}
}

// NewNodeFromConfig builds a native/sim transport using the BLE factory and returns a node.
func NewNodeFromConfig(cfg *core.Config) (*Node, error) {
	local := core.PeerInfo{
		ID:      cfg.Device.Name,
		Name:    cfg.Device.Name,
		Address: "", // filled by transport-specific logic when available
		Role:    cfg.Device.Role,
		OS:      cfg.Device.OS,
		Arch:    cfg.Device.Arch,
		Tags:    cfg.Device.Tags,
	}
	return NewNode(nil, &cfg.Network, local), nil
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
	return nil
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
