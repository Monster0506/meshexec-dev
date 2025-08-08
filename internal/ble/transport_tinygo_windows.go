//go:build windows && !goble

package ble

import (
    "context"
    "errors"
    "strings"
    "time"

    bt "tinygo.org/x/bluetooth"

    core "github.com/monster0506/meshexec/internal"
)

// tgTransport implements BLETransport using tinygo bluetooth on Windows.
// Windows currently supports central mode (scan/connect) but not peripheral
// advertisement/services via this library. Advertise and CreateGATTService
// will return an error.
type tgTransport struct {
    adapter *bt.Adapter
}

// NewNativeTransport returns a tinygo-backed transport on Windows.
func NewNativeTransport(_ *core.NetworkConfig) (core.BLETransport, error) {
    ad := bt.DefaultAdapter
    if err := ad.Enable(); err != nil {
        return nil, err
    }
    return &tgTransport{adapter: ad}, nil
}

func (t *tgTransport) Advertise(ctx context.Context, serviceData []byte) error {
    return errors.New("advertising not supported on Windows backend (tinygo bluetooth)")
}

func (t *tgTransport) Scan(ctx context.Context) (<-chan *core.Advertisement, error) {
    out := make(chan *core.Advertisement, 32)
    go func() {
        defer close(out)
        _ = t.adapter.Scan(func(_ *bt.Adapter, dev bt.ScanResult) {
            adv := &core.Advertisement{
                Address:   strings.ToLower(dev.Address.String()),
                Name:      dev.LocalName(),
                ServiceData: map[string][]byte{},
                RSSI:      int(dev.RSSI),
                Timestamp: time.Now(),
            }
            select { case out <- adv: default: }
        })
        <-ctx.Done()
        _ = t.adapter.StopScan()
    }()
    return out, nil
}

func (t *tgTransport) Connect(ctx context.Context, addr string) (*core.Connection, error) {
    // Parse address
    var macAddr bt.MACAddress
    macAddr.Set(addr)
    dev, err := t.adapter.Connect(bt.Address{MACAddress: macAddr}, bt.ConnectionParams{})
    if err != nil {
        return nil, err
    }
    // Best-effort timeout context to ensure we honor ctx cancellation
    go func() {
        select {
        case <-ctx.Done():
            _ = dev.Disconnect()
        }
    }()
    // MTU retrieval is characteristic-specific in tinygo API; use a reasonable default
    return &core.Connection{Address: strings.ToLower(addr), MTU: 185, Connected: true}, nil
}

func (t *tgTransport) CreateGATTService() (*core.GATTService, error) {
    return nil, errors.New("GATT server not supported on Windows backend (tinygo bluetooth)")
}

