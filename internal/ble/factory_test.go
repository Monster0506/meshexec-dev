//go:build ble

package ble

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

// getPlatform/fileExists/newSim helpers
func TestFactory_GetPlatformAndFileExists(t *testing.T) {
	p := getPlatform()
	if p == "" {
		t.Fatalf("getPlatform returned empty")
	}

	dir := t.TempDir()
	f := filepath.Join(dir, "x.tmp")
	if err := os.WriteFile(f, []byte("hi"), 0644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	if !fileExists(f) {
		t.Fatalf("expected fileExists true for %s", f)
	}
}

func TestFactory_NewSim_AppliesInterval(t *testing.T) {
	cfg := &core.NetworkConfig{AdvertiseInterval: 150}
	tr := newSim(cfg, logging.NewLogger("none"))
	if bt, ok := tr.(*Transport); ok {
		if got := bt.advertiseInterval.Milliseconds(); got != 150 {
			t.Fatalf("expected interval 150ms, got %d", got)
		}
	} else {
		t.Fatalf("expected *Transport type from newSim")
	}
}

func TestFactory_GetActualTransportType(t *testing.T) {
	got := getActualTransportType()
	if got == "" {
		t.Fatalf("expected non-empty transport type on %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

// NewWithLogger factory paths (auto/sim)
func TestFactory_NewWithLogger_AutoSimPath_Basics(t *testing.T) {
	t.Setenv("MESHEXEC_BLE_IMPL", "sim")
	tr, err := NewWithLogger(&core.NetworkConfig{}, logging.NewLogger("none"))
	if err != nil || tr == nil {
		t.Fatalf("expected sim transport: %v", err)
	}
}

func TestFactory_NewWithLogger_AutoDefault_Basics(t *testing.T) {
	t.Setenv("MESHEXEC_BLE_IMPL", "")
	_, _ = NewWithLogger(&core.NetworkConfig{}, logging.NewLogger("none"))
}
