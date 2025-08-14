package mesh

import (
	"context"
	"testing"
	"time"

	core "github.com/monster0506/meshexec/internal"
)

// stub transport removed; Node tests operate locally without transport

func TestNode_StartStop_SubscribeAndSend(t *testing.T) {
	cfg := core.DefaultConfig()
	n := NewNode(nil, &cfg.Network, core.PeerInfo{ID: "self", Name: "self"})

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

// Fragmentation framing unit tests (local only; transport not used)
func TestNode_BuildFramesAndReassemble(t *testing.T) {
	cfg := core.DefaultConfig()
	n := NewNode(nil, &cfg.Network, core.PeerInfo{ID: "self", Name: "self"})
	// Create payload large enough to force multiple frames
	big := make([]byte, 0, 1)
	for i := 0; i < 400; i++ {
		big = append(big, byte('A'+(i%26)))
	}
	frames := n.buildFramesWithMTU("msg-1", big, 120)
	if len(frames) < 2 {
		t.Fatalf("expected fragmentation, got %d frames", len(frames))
	}
	// Feed frames in order and expect final reassembly only at end
	for i, fr := range frames {
		full := n.tryReassemble(fr)
		if i < len(frames)-1 {
			if full != nil {
				t.Fatalf("expected nil before last frame, got %d bytes", len(full))
			}
		} else {
			if full == nil || len(full) != len(big) {
				t.Fatalf("expected full reassembly of %d bytes, got %v", len(big), len(full))
			}
		}
	}
}

func TestNode_GetPeers_Empty(t *testing.T) {
	cfg := core.DefaultConfig()
	n := NewNode(nil, &cfg.Network, core.PeerInfo{ID: "self", Name: "self"})
	peers := n.GetPeers()
	if len(peers) != 0 {
		t.Fatalf("expected no peers, got %d", len(peers))
	}
}
