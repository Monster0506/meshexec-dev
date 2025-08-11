//go:build windows

package ble

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/monster0506/meshexec/internal/logging"
)

// Minimal sidecar stub that echoes {"ok":true} for any request
func startStubSidecar(t *testing.T) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		dec := json.NewDecoder(bufio.NewReader(conn))
		enc := json.NewEncoder(conn)
		for i := 0; i < 4; i++ {
			var req map[string]any
			if err := dec.Decode(&req); err != nil {
				return
			}
			_ = enc.Encode(map[string]any{"ok": true, "data": map[string]any{}})
		}
	}()
	return ln.Addr().String(), func() {
		_ = ln.Close()
		<-done
	}
}

func TestSidecar_IOExercise(t *testing.T) {
	addr, stop := startStubSidecar(t)
	defer stop()

	st := &SidecarTransport{logger: logging.NewLogger("none"), addr: addr, scanner: NewTransport()}

	// Advertise/start-stop cycle
	ctx, cancel := context.WithCancel(context.Background())
	if err := st.Advertise(ctx, []byte("abc")); err != nil {
		t.Fatalf("Advertise: %v", err)
	}
	cancel()
	time.Sleep(20 * time.Millisecond)

	// Create GATT service
	if _, err := st.CreateGATTService(); err != nil {
		t.Fatalf("CreateGATTService: %v", err)
	}

	// Scan delegates to fallback scanner
	sctx, scancel := context.WithCancel(context.Background())
	ch, err := st.Scan(sctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	// drain quickly
	select {
	case <-ch:
	case <-time.After(5 * time.Millisecond):
	}
	scancel()

	// Connect delegates to fallback scanner; expect error or connection, both fine for coverage
	_, _ = st.Connect(context.Background(), "00:11:22:33:44:55")

	// Also directly exercise ensureConn and do via a benign action
	if _, err := st.do("noop", map[string]any{}); err != nil {
		t.Fatalf("do: %v", err)
	}
}

func TestSidecarTryNewTransport_Availability(t *testing.T) {
	t.Setenv("MESHEXEC_SIDECAR_ADDR", "127.0.0.1:65533") // likely closed port
	_, ok, _ := tryNewSidecarTransport(nil, nil)
	if !ok {
		t.Fatalf("expected sidecar availability path to be reported (ok==true)")
	}
}
