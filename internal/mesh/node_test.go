package mesh

import (
    "context"
    "testing"
    "time"

    core "github.com/monster0506/meshexec/internal"
)

// stubTransport implements core.BLETransport minimally for tests
type stubTransport struct{}

func (s stubTransport) Advertise(ctx context.Context, serviceData []byte) error { return nil }
func (s stubTransport) Scan(ctx context.Context) (<-chan *core.Advertisement, error) {
    ch := make(chan *core.Advertisement)
    close(ch)
    return ch, nil
}
func (s stubTransport) Connect(ctx context.Context, addr string) (*core.Connection, error) {
    return &core.Connection{Address: addr, MTU: 185, Connected: true}, nil
}
func (s stubTransport) CreateGATTService() (*core.GATTService, error) { return &core.GATTService{UUID: "x"}, nil }

func TestNode_StartStop_SubscribeAndSend(t *testing.T) {
    cfg := core.DefaultConfig()
    n := NewNode(stubTransport{}, &cfg.Network, core.PeerInfo{ID: "self", Name: "self"})

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    if err := n.Start(ctx); err != nil {
        t.Fatalf("start error: %v", err)
    }

    ch := n.Subscribe(core.MessageTypeCommand)
    msg := &core.MeshMessage{ID: "1", TTL: 5, Sender: "self", Target: []string{"all"}, Type: core.MessageTypeCommand, Timestamp: time.Now().Unix()}
    if err := n.SendMessage(msg); err != nil {
        t.Fatalf("send error: %v", err)
    }
    select {
    case got := <-ch:
        if got == nil || got.ID != msg.ID {
            t.Fatalf("unexpected message: %+v", got)
        }
    case <-time.After(500 * time.Millisecond):
        t.Fatal("timeout waiting for message")
    }

    if err := n.Stop(); err != nil {
        t.Fatalf("stop error: %v", err)
    }
}


