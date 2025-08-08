//go:build linux || darwin

package ble

import (
	"context"
	"errors"
	"sync"
	"time"

	goble "github.com/go-ble/ble"
	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

// nativeTransport is a go-ble backed implementation of core.BLETransport.
type nativeTransport struct {
	device goble.Device
	cfg    *core.NetworkConfig
	logger *logging.Logger

	// cached service/characteristic for future use
	svc      *goble.Service
	chr      *goble.Characteristic
	mu       sync.RWMutex
	chrValue []byte
}

// NewNativeTransport initializes the platform BLE device and returns a transport.
func NewNativeTransport(cfg *core.NetworkConfig) (core.BLETransport, error) {
	return newNativeWithLogger(cfg, nil)
}

// newNativeWithLogger initializes the platform BLE device with a logger.
func newNativeWithLogger(cfg *core.NetworkConfig, logger *logging.Logger) (core.BLETransport, error) {
	if logger == nil {
		logger = logging.NewLogger("info")
	}

	logger.Info("Initializing native BLE transport", nil)

	dev, err := newDevice()
	if err != nil {
		logger.Error("Failed to initialize BLE device", err, nil)
		return nil, err
	}

	goble.SetDefaultDevice(dev)

	transport := &nativeTransport{
		device: dev,
		cfg:    cfg,
		logger: logger,
	}

	logger.Info("Native BLE transport initialized successfully", map[string]interface{}{
		"device_type": "go-ble",
	})

	return transport, nil
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

	t.logger.Info("Starting native BLE advertisement", map[string]interface{}{
		"service_uuid":        su,
		"device_name":         name,
		"service_data_length": len(serviceData),
	})

	// This call blocks until ctx is canceled
	err := goble.AdvertiseNameAndServices(ctx, name, svcUUID)
	if err != nil {
		t.logger.Error("Native BLE advertisement failed", err, map[string]interface{}{
			"service_uuid": su,
		})
	} else {
		t.logger.Info("Native BLE advertisement stopped", map[string]interface{}{
			"service_uuid": su,
		})
	}

	return err
}

// Scan performs a BLE scan and forwards advertisements to a channel.
func (t *nativeTransport) Scan(ctx context.Context) (<-chan *core.Advertisement, error) {
	out := make(chan *core.Advertisement, 32)

	t.logger.Info("Starting native BLE scan", nil)

	go func() {
		defer close(out)
		defer t.logger.Info("Native BLE scan stopped", nil)

		scanCount := 0
		// Updated Scan call with new API: add nil as AdvFilter parameter
		_ = goble.Scan(ctx, true, func(a goble.Advertisement) {
			scanCount++
			sd := map[string][]byte{}
			// Updated ServiceData call: no longer takes UUID parameter
			serviceData := a.ServiceData()
			for _, data := range serviceData {
				sd[data.UUID.String()] = append([]byte(nil), data.Data...)
			}
			adv := &core.Advertisement{
				Address:     a.Addr().String(),
				Name:        a.LocalName(),
				ServiceData: sd,
				RSSI:        a.RSSI(),
				Timestamp:   time.Now(),
			}

			t.logger.Debug("Native BLE advertisement received", map[string]interface{}{
				"address":        adv.Address,
				"name":           adv.Name,
				"rssi":           adv.RSSI,
				"services_count": len(a.Services()),
				"scan_count":     scanCount,
			})

			select {
			case out <- adv:
			default:
				t.logger.Warn("Dropped BLE advertisement - slow receiver", map[string]interface{}{
					"address": adv.Address,
				})
			}
		}, nil) // Added nil as AdvFilter parameter

		t.logger.Info("Native BLE scan completed", map[string]interface{}{
			"total_advertisements": scanCount,
		})
	}()

	return out, nil
}

// Connect dials a remote BLE peripheral.
func (t *nativeTransport) Connect(ctx context.Context, addr string) (*core.Connection, error) {
	if addr == "" {
		t.logger.Error("Connection attempt with empty address", nil, nil)
		return nil, errors.New("addr is required")
	}

	t.logger.Info("Attempting native BLE connection", map[string]interface{}{
		"target_address": addr,
	})

	// Updated Dial call: convert string to ble.Addr
	addrObj := goble.NewAddr(addr)
	c, err := goble.Dial(ctx, addrObj)
	if err != nil {
		t.logger.Error("Native BLE connection failed", err, map[string]interface{}{
			"target_address": addr,
		})
		return nil, err
	}

	// Updated MTU handling: AttMTU() method no longer exists
	mtu := 0
	if c != nil {
		// Use default MTU since AttMTU() is no longer available
		mtu = 23 // Default BLE MTU
	}

	t.logger.Info("Native BLE connection established", map[string]interface{}{
		"address": addr,
		"mtu":     mtu,
	})

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

	t.logger.Info("Creating native GATT service", map[string]interface{}{
		"service_uuid":        su,
		"characteristic_uuid": cu,
	})

	svc := goble.NewService(goble.MustParse(su))
	chr := svc.NewCharacteristic(goble.MustParse(cu))

	// Provide a static value for now to support read requests
	t.mu.Lock()
	t.chrValue = []byte("ready")
	t.mu.Unlock()
	chr.SetValue([]byte("ready"))

	if err := goble.AddService(svc); err != nil {
		t.logger.Error("Failed to add GATT service", err, map[string]interface{}{
			"service_uuid": su,
		})
		return nil, err
	}

	t.mu.Lock()
	t.svc = svc
	t.chr = chr
	t.mu.Unlock()

	t.logger.Info("Native GATT service created successfully", map[string]interface{}{
		"service_uuid":        su,
		"characteristic_uuid": cu,
	})

	return &core.GATTService{
		UUID: su,
		Characteristics: []core.GATTCharacteristic{
			{UUID: cu},
		},
	}, nil
}
