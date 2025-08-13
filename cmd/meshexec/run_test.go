package main

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

// startTestTCPServer starts a minimal loopback server that accepts one JSON {"cmd":"..."}
// and returns a canned result. It closes after one response.
func startTestTCPServer(t *testing.T) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = c.Close() }()
		var req map[string]string
		_ = json.NewDecoder(c).Decode(&req)
		_ = json.NewEncoder(c).Encode(map[string]any{"ok": true, "result": map[string]any{"status": "success", "exit_code": 0, "stdout": "ok", "stderr": ""}})
	}()
	return ln.Addr().String(), func() { _ = ln.Close(); <-done }
}

func TestSendCommandTCP_CannedResponse(t *testing.T) {
	addr, stop := startTestTCPServer(t)
	defer stop()
	res, err := sendCommandTCP(addr, "echo hi", 2*time.Second)
	if err != nil {
		t.Fatalf("send error: %v", err)
	}
	if res == nil || res.Status != "success" || res.ExitCode != 0 {
		t.Fatalf("bad result: %+v", res)
	}
}
