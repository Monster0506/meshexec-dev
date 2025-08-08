package ble

import (
	"context"
	"testing"
	"time"

	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

type mockTransport struct {
	scanCh      chan *core.Advertisement
	connections map[string]*core.Connection
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		scanCh:      make(chan *core.Advertisement, 8),
		connections: make(map[string]*core.Connection),
	}
}

func (m *mockTransport) Advertise(ctx context.Context, serviceData []byte) error {
	// Simulate a long-running advertise until ctx is cancelled
	go func() { <-ctx.Done() }()
	return nil
}

func (m *mockTransport) Scan(ctx context.Context) (<-chan *core.Advertisement, error) {
	out := make(chan *core.Advertisement, 8)
	// Forward from internal scanCh to returned channel
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case adv, ok := <-m.scanCh:
				if !ok {
					return
				}
				select {
				case out <- adv:
				default:
				}
			}
		}
	}()
	return out, nil
}

func (m *mockTransport) Connect(ctx context.Context, addr string) (*core.Connection, error) {
	if c, ok := m.connections[addr]; ok {
		c.Connected = true
		return c, nil
	}
	c := &core.Connection{Address: addr, MTU: 185, Connected: true}
	m.connections[addr] = c
	return c, nil
}

func (m *mockTransport) CreateGATTService() (*core.GATTService, error) {
	return &core.GATTService{UUID: "test", Characteristics: []core.GATTCharacteristic{{UUID: "char"}}}, nil
}

func TestManagerStartDiscoveryAndUpdates(t *testing.T) {
	mt := newMockTransport()
	logger := logging.NewLogger("error") // Use error level to reduce test noise
	mgr := NewManager(mt, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := mgr.StartDiscovery(ctx); err != nil {
		t.Fatalf("StartDiscovery error: %v", err)
	}

	subCtx, subCancel := context.WithCancel(context.Background())
	defer subCancel()
	updates := mgr.Subscribe(subCtx)

	// Emit an advertisement
	adv := &core.Advertisement{Address: "AA:BB:CC:DD:EE:FF", Name: "peer1", RSSI: -50, Timestamp: time.Now()}
	mt.scanCh <- adv

	select {
	case p := <-updates:
		if p.Address != adv.Address || p.Name != adv.Name {
			t.Fatalf("unexpected peer: got %+v", p)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for peer update")
	}

	// Ensure ListPeers contains the peer
	peers := mgr.ListPeers()
	if len(peers) != 1 || peers[0].Address != adv.Address {
		t.Fatalf("expected 1 peer with address %s, got %+v", adv.Address, peers)
	}
}

func TestManagerConnectUpdatesPeer(t *testing.T) {
	mt := newMockTransport()
	logger := logging.NewLogger("error") // Use error level to reduce test noise
	mgr := NewManager(mt, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := mgr.StartDiscovery(ctx); err != nil {
		t.Fatalf("StartDiscovery error: %v", err)
	}

	addr := "11:22:33:44:55:66"
	// Seed a discovered peer
	mt.scanCh <- &core.Advertisement{Address: addr, Name: "peer2", RSSI: -42, Timestamp: time.Now()}

	// Give manager a moment to process
	time.Sleep(50 * time.Millisecond)

	if _, err := mgr.Connect(ctx, addr); err != nil {
		t.Fatalf("Connect error: %v", err)
	}

	// Verify peer connected flag
	p, ok := mgr.GetPeer(addr)
	if !ok {
		t.Fatalf("peer not found after connect")
	}
	if !p.Connected {
		t.Fatalf("expected peer Connected=true, got %+v", p)
	}
}
