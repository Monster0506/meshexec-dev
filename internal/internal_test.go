package internal

import "testing"

func TestDefaultConfig_AndMeshExecErrorErrorMethod(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil || cfg.Device.Name == "" || cfg.Network.ServiceUUID == "" {
		t.Fatalf("unexpected default config: %+v", cfg)
	}

	me := &MeshExecError{Type: ErrorTypeExecution, Message: "boom", Code: "x"}
	if me.Error() != "boom" {
		t.Fatalf("unexpected error string: %q", me.Error())
	}
}
