package main

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"
)

func TestSendCommandTCP_UsesDialSeam(t *testing.T) {
	old := tcpDial
	defer func() { tcpDial = old }()

	server, client := net.Pipe()
	defer func() { _ = server.Close(); _ = client.Close() }()
	tcpDial = func(addr string, timeout time.Duration) (net.Conn, error) { return client, nil }

	go func() {
		rd := bufio.NewReader(server)
		_, _ = rd.ReadString('\n')
		_, _ = server.Write([]byte("{\"ok\":true,\"result\":{\"status\":\"success\",\"exit_code\":0,\"stdout\":\"ok\",\"stderr\":\"\"}}\n"))
	}()

	res, err := sendCommandTCP("ignored", "echo hi", 2*time.Second)
	if err != nil {
		t.Fatalf("send error: %v", err)
	}
	if res == nil || res.Status != "success" || res.ExitCode != 0 || strings.TrimSpace(res.Stdout) != "ok" {
		t.Fatalf("bad result: %+v", res)
	}
}
