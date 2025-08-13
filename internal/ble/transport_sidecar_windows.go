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

	// service/characteristic identifiers for central operations
	serviceUUID string
	charUUID    string
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
	svc := cfg.ServiceUUID
	chr := cfg.CharacteristicUUID
	st := &SidecarTransport{logger: logger, addr: addr, scanner: fallback, serviceUUID: svc, charUUID: chr}
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

// DoSidecar exposes the sidecar request/response for higher-level operations (e.g., GATT write/notify).
func (t *SidecarTransport) DoSidecar(action string, params map[string]interface{}) (*sidecarResponse, error) {
	return t.do(action, params)
}

func (t *SidecarTransport) Advertise(ctx context.Context, serviceData []byte) error {
	if t.logger != nil {
		t.logger.Info("Sidecar: starting advertise", map[string]interface{}{"len": len(serviceData)})
	}
	params := map[string]interface{}{
		"service_uuid":     t.serviceUUID,
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
	params := map[string]interface{}{
		"service_uuid":        t.serviceUUID,
		"characteristic_uuid": t.charUUID,
		"properties":          "read,write,notify",
	}
	if _, err := t.do("gatt_create", params); err != nil {
		return nil, err
	}
	return &core.GATTService{UUID: t.serviceUUID, Characteristics: []core.GATTCharacteristic{{UUID: t.charUUID, Writable: true}}}, nil
}

// SubscribeWriteNotifications subscribes to incoming GATT write events from sidecar and streams payloads.
// A dedicated TCP connection is used for the subscription so it doesn't interfere with request/response traffic.
func (t *SidecarTransport) SubscribeWriteNotifications(ctx context.Context) (<-chan []byte, func(), error) {
	// open dedicated connection
	c, err := net.Dial("tcp", t.addr)
	if err != nil {
		return nil, nil, err
	}
	out := make(chan []byte, 32)
	// send subscribe request
	enc := json.NewEncoder(c)
	subReq := sidecarRequest{Action: "gatt_subscribe"}
	if err := enc.Encode(&subReq); err != nil {
		_ = c.Close()
		return nil, nil, err
	}
	// read responses/events
	go func() {
		defer close(out)
		dec := json.NewDecoder(bufio.NewReader(c))
		for {
			select {
			case <-ctx.Done():
				// attempt graceful unsubscribe
				_ = json.NewEncoder(c).Encode(sidecarRequest{Action: "gatt_unsubscribe"})
				_ = c.Close()
				return
			default:
			}
			var resp sidecarResponse
			if err := dec.Decode(&resp); err != nil {
				_ = c.Close()
				return
			}
			if !resp.Ok {
				continue
			}
			if resp.Data != nil {
				if evt, ok := resp.Data["event"].(string); ok && evt == "gatt_write" {
					if v, ok := resp.Data["value_b64"].(string); ok {
						if b, err := base64.StdEncoding.DecodeString(v); err == nil {
							select {
							case out <- b:
							default:
							}
						}
					}
				}
			}
		}
	}()
	// unsubscribe func closes connection and channel will end
	unsub := func() { _ = c.Close() }
	return out, unsub, nil
}

// SendNotification sends a notification payload via sidecar GATT characteristic.
func (t *SidecarTransport) SendNotification(ctx context.Context, data []byte) error {
	if t.logger != nil {
		t.logger.Debug("Sidecar: sending GATT notification", map[string]interface{}{"len": len(data)})
	}
	p := map[string]interface{}{
		"value_b64": base64.StdEncoding.EncodeToString(data),
	}
	// Retry with simple backoff
	var lastErr error
	backoff := 50 * time.Millisecond
	for attempt := 0; attempt < 4; attempt++ {
		if ctx.Err() != nil {
			if lastErr != nil {
				return lastErr
			}
			return ctx.Err()
		}
		resp, err := t.DoSidecar("gatt_notify", p)
		if err == nil && resp.Ok {
			return nil
		}
		if err != nil {
			lastErr = err
		} else if !resp.Ok {
			if resp.Error == "" {
				resp.Error = "gatt_notify failed"
			}
			lastErr = errors.New(resp.Error)
		}
		select {
		case <-time.After(backoff):
			if backoff < 400*time.Millisecond {
				backoff *= 2
			}
		case <-ctx.Done():
			if lastErr != nil {
				return lastErr
			}
			return ctx.Err()
		}
	}
	if lastErr == nil {
		lastErr = errors.New("gatt_notify failed after retries")
	}
	return lastErr
}

// EffectiveMTU returns the effective ATT_MTU for notifications.
// Windows GATT typically negotiates ~185; we use a conservative default.
func (t *SidecarTransport) EffectiveMTU() int { return 185 }

// CentralBroadcast writes the given payload to all discovered peers advertising
// the configured service UUID by connecting as a central and performing a characteristic write.
// This provides a simple one-hop broadcast suitable for command delivery.
func (t *SidecarTransport) CentralBroadcast(ctx context.Context, data []byte) error {
	if t.serviceUUID == "" || t.charUUID == "" {
		return errors.New("missing service/characteristic UUID for central broadcast")
	}
	p := map[string]interface{}{
		"service_uuid":        t.serviceUUID,
		"characteristic_uuid": t.charUUID,
		"value_b64":           base64.StdEncoding.EncodeToString(data),
		"scan_ms":             2000,
	}
	// Best-effort: make a short-lived request; sidecar performs scan/connect/write internally
	if _, err := t.do("central_broadcast", p); err != nil {
		return err
	}
	return nil
}
