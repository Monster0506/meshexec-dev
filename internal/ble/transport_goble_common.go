//go:build linux || darwin

package ble

import (
    "context"
    "errors"
    "sync"
    "time"

    goble "github.com/go-ble/ble"
    core "github.com/monster0506/meshexec/internal"
)

// nativeTransport is a go-ble backed implementation of core.BLETransport.
type nativeTransport struct {
    device    goble.Device
    cfg       *core.NetworkConfig

    // cached service/characteristic for future use
    svc       *goble.Service
    chr       *goble.Characteristic
    mu        sync.RWMutex
    chrValue  []byte
}

// NewNativeTransport initializes the platform BLE device and returns a transport.
func NewNativeTransport(cfg *core.NetworkConfig) (core.BLETransport, error) {
    dev, err := newDevice()
    if err != nil {
        return nil, err
    }
    goble.SetDefaultDevice(dev)
    return &nativeTransport{device: dev, cfg: cfg}, nil
}

// Advertise starts advertising the local name and a single service UUID.
// The serviceData parameter is not fully utilized due to library constraints,
// but can be mapped to manufacturer data in future iterations.
func (t *nativeTransport) Advertise(ctx context.Context, serviceData []byte) error {
    // Use configured service UUID if present; otherwise defaults
    su := ""
    if t.cfg != nil && t.cfg.ServiceUUID != "" {
        su = t.cfg.ServiceUUID
    } else {
        su = core.DefaultConfig().Network.ServiceUUID
    }
    svcUUID := goble.MustParse(su)
    name := "meshexec"
    // This call blocks until ctx is canceled
    return goble.AdvertiseNameAndServices(ctx, name, svcUUID)
}

// Scan performs a BLE scan and forwards advertisements to a channel.
func (t *nativeTransport) Scan(ctx context.Context) (<-chan *core.Advertisement, error) {
    out := make(chan *core.Advertisement, 32)

    go func() {
        defer close(out)
        _ = goble.Scan(ctx, true, func(a goble.Advertisement) {
            sd := map[string][]byte{}
            // Map service data if available
            for _, u := range a.Services() {
                if b := a.ServiceData(u); len(b) > 0 {
                    sd[u.String()] = append([]byte(nil), b...)
                }
            }
            adv := &core.Advertisement{
                Address:   a.Addr().String(),
                Name:      a.LocalName(),
                ServiceData: sd,
                RSSI:      a.RSSI(),
                Timestamp: time.Now(),
            }
            select {
            case out <- adv:
            default:
                // drop if receiver is slow
            }
        })
    }()

    return out, nil
}

// Connect dials a remote BLE peripheral.
func (t *nativeTransport) Connect(ctx context.Context, addr string) (*core.Connection, error) {
    if addr == "" {
        return nil, errors.New("addr is required")
    }
    c, err := goble.Dial(ctx, addr)
    if err != nil {
        return nil, err
    }
    mtu := 0
    if c != nil {
        mtu = c.AttMTU()
    }
    return &core.Connection{Address: addr, MTU: mtu, Connected: true}, nil
}

// CreateGATTService returns a placeholder description; real service wiring will be
// added in the next step when sending/receiving over characteristics is implemented.
func (t *nativeTransport) CreateGATTService() (*core.GATTService, error) {
    // Resolve UUIDs from config or defaults
    su := ""
    cu := ""
    if t.cfg != nil {
        su = t.cfg.ServiceUUID
        cu = t.cfg.CharacteristicUUID
    }
    if su == "" {
        su = core.DefaultConfig().Network.ServiceUUID
    }
    if cu == "" {
        cu = core.DefaultConfig().Network.CharacteristicUUID
    }

    svc := goble.NewService(goble.MustParse(su))
    chr := svc.NewCharacteristic(goble.MustParse(cu))

    // Provide a static value for now to support read requests
    t.mu.Lock()
    t.chrValue = []byte("ready")
    t.mu.Unlock()
    chr.SetValue([]byte("ready"))

    if err := goble.AddService(svc); err != nil {
        return nil, err
    }

    t.mu.Lock()
    t.svc = svc
    t.chr = chr
    t.mu.Unlock()

    return &core.GATTService{
        UUID: su,
        Characteristics: []core.GATTCharacteristic{
            {UUID: cu},
        },
    }, nil
}

