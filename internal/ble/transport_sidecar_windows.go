//go:build windows

package ble

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

// SidecarTransport proxies BLE operations to an external sidecar process over TCP.
// It implements Advertise and CreateGATTService via sidecar; Scan uses the existing TinyGo or sim transport as fallback.
type SidecarTransport struct {
	logger *logging.Logger

	mu   sync.Mutex
	conn net.Conn
	addr string

	// fallback scanner for discovery
	scanner core.BLETransport
}

type sidecarRequest struct {
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params,omitempty"`
}
type sidecarResponse struct {
	Ok    bool                   `json:"ok"`
	Error string                 `json:"error,omitempty"`
	Data  map[string]interface{} `json:"data,omitempty"`
}

func tryNewSidecarTransport(cfg *core.NetworkConfig, logger *logging.Logger) (core.BLETransport, bool, error) {
	if logger == nil {
		logger = logging.NewLogger("info")
	}
	addr := os.Getenv("MESHEXEC_SIDECAR_ADDR")
	if strings.TrimSpace(addr) == "" {
		addr = "127.0.0.1:8765"
	}
	// Attempt to connect quickly; if cannot, report available but error
	conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
	if err != nil {
		return nil, true, fmt.Errorf("sidecar not reachable at %s: %w", addr, err)
	}
	_ = conn.Close()

	// Build fallback scanner: prefer native tinygo path, else sim (avoid recursion into factory)
	fallback, err := newNativeWithLogger(cfg, logger)
	if err != nil {
		fallback = newSim(cfg, logger)
	}
	st := &SidecarTransport{logger: logger, addr: addr, scanner: fallback}
	return st, true, nil
}

func (t *SidecarTransport) ensureConn() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn != nil {
		return nil
	}
	c, err := net.Dial("tcp", t.addr)
	if err != nil {
		return err
	}
	t.conn = c
	return nil
}

func (t *SidecarTransport) do(action string, params map[string]interface{}) (*sidecarResponse, error) {
	if err := t.ensureConn(); err != nil {
		return nil, err
	}
	req := sidecarRequest{Action: action, Params: params}
	enc := json.NewEncoder(t.conn)
	if err := enc.Encode(&req); err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bufio.NewReader(t.conn))
	var resp sidecarResponse
	if err := dec.Decode(&resp); err != nil {
		return nil, err
	}
	if !resp.Ok {
		if resp.Error == "" {
			resp.Error = "unknown sidecar error"
		}
		return nil, errors.New(resp.Error)
	}
	return &resp, nil
}

func (t *SidecarTransport) Advertise(ctx context.Context, serviceData []byte) error {
	if t.logger != nil {
		t.logger.Info("Sidecar: starting advertise", map[string]interface{}{"len": len(serviceData)})
	}
	cfg := core.DefaultConfig()
	params := map[string]interface{}{
		"service_uuid":     cfg.Network.ServiceUUID,
		"local_name":       "meshexec",
		"service_data_b64": base64.StdEncoding.EncodeToString(serviceData),
	}
	if _, err := t.do("advertise_start", params); err != nil {
		return err
	}
	// Stop when context done
	go func() {
		<-ctx.Done()
		_, _ = t.do("advertise_stop", nil)
	}()
	return nil
}

func (t *SidecarTransport) Scan(ctx context.Context) (<-chan *core.Advertisement, error) {
	if t.scanner == nil {
		return nil, fmt.Errorf("no fallback scanner available")
	}
	return t.scanner.Scan(ctx)
}

func (t *SidecarTransport) Connect(ctx context.Context, addr string) (*core.Connection, error) {
	// Delegate to fallback scanner transport if it supports connect
	if t.scanner == nil {
		return &core.Connection{Address: addr, MTU: 185, Connected: false}, nil
	}
	return t.scanner.Connect(ctx, addr)
}

func (t *SidecarTransport) CreateGATTService() (*core.GATTService, error) {
	if t.logger != nil {
		t.logger.Info("Sidecar: creating GATT service", nil)
	}
	cfg := core.DefaultConfig()
	params := map[string]interface{}{
		"service_uuid":        cfg.Network.ServiceUUID,
		"characteristic_uuid": cfg.Network.CharacteristicUUID,
		"properties":          "read,write,notify",
	}
	if _, err := t.do("gatt_create", params); err != nil {
		return nil, err
	}
	return &core.GATTService{UUID: cfg.Network.ServiceUUID, Characteristics: []core.GATTCharacteristic{{UUID: cfg.Network.CharacteristicUUID, Writable: true}}}, nil
}
