package internal

import (
	"errors"
	"testing"
)

func TestNewErrorConstructors(t *testing.T) {
	e1 := NewNetworkError("adapter_unavailable", "no adapter", map[string]interface{}{"x": 1})
	if e1.Type != ErrorTypeNetwork || e1.Code != "adapter_unavailable" || e1.Message != "no adapter" || e1.Details["x"].(int) != 1 {
		t.Fatalf("unexpected NewNetworkError: %+v", e1)
	}
	e2 := NewExecutionError("dangerous_command", "blocked", nil)
	if e2.Type != ErrorTypeExecution || e2.Code != "dangerous_command" || e2.Message != "blocked" || e2.Details != nil {
		t.Fatalf("unexpected NewExecutionError: %+v", e2)
	}
	e3 := NewSecurityError("signature_invalid", "invalid", map[string]interface{}{"why": "sig"})
	if e3.Type != ErrorTypeSecurity || e3.Code != "signature_invalid" || e3.Details["why"].(string) != "sig" {
		t.Fatalf("unexpected NewSecurityError: %+v", e3)
	}
	e4 := NewConfigError("invalid_config", "bad", nil)
	if e4.Type != ErrorTypeConfig || e4.Code != "invalid_config" {
		t.Fatalf("unexpected NewConfigError: %+v", e4)
	}
	e5 := NewTargetingError("parse_error", "bad expr", nil)
	if e5.Type != ErrorTypeTargeting || e5.Code != "parse_error" {
		t.Fatalf("unexpected NewTargetingError: %+v", e5)
	}
}

func TestWrapAsAndFromError(t *testing.T) {
	base := errors.New("root cause")
	w := WrapAs(base, ErrorTypeNetwork, "scan_failed", "scan failed", map[string]interface{}{"op": "scan"})
	if w.Type != ErrorTypeNetwork || w.Code != "scan_failed" || w.Message != "scan failed" || w.Details["op"].(string) != "scan" {
		t.Fatalf("unexpected WrapAs from std error: %+v", w)
	}

	// Wrapping an existing MeshExecError should preserve unless overridden
	original := &MeshExecError{Type: ErrorTypeExecution, Code: "x", Message: "y", Details: map[string]interface{}{"a": 1}}
	w2 := WrapAs(original, "", "", "", map[string]interface{}{"b": 2})
	if w2.Type != ErrorTypeExecution || w2.Code != "x" || w2.Message != "y" || w2.Details["a"].(int) != 1 || w2.Details["b"].(int) != 2 {
		t.Fatalf("unexpected WrapAs preserve: %+v", w2)
	}
	// Override fields
	w3 := WrapAs(original, ErrorTypeSecurity, "new", "msg", nil)
	if w3.Type != ErrorTypeSecurity || w3.Code != "new" || w3.Message != "msg" {
		t.Fatalf("unexpected WrapAs override: %+v", w3)
	}

	// FromError
	if fe := FromError(nil); fe != nil {
		t.Fatalf("expected nil for nil error")
	}
	if fe := FromError(original); fe != original {
		t.Fatalf("expected passthrough for MeshExecError")
	}
	if fe := FromError(base); fe.Type != ErrorTypeExecution || fe.Code != "generic_error" || fe.Message != base.Error() {
		t.Fatalf("unexpected FromError std: %+v", fe)
	}
}

func TestFormatUserError(t *testing.T) {
	if s := FormatUserError(nil); s != "" {
		t.Fatalf("expected empty string for nil error")
	}
	e := NewNetworkError("adapter_unavailable", "scan failed: adapter unavailable", nil)
	out := FormatUserError(e)
	if out == "" || out[0] != '[' {
		t.Fatalf("unexpected format: %q", out)
	}
	e2 := NewExecutionError("dangerous_command", "blocked by safety", nil)
	out2 := FormatUserError(e2)
	if !contains(out2, "dangerous_command") {
		t.Fatalf("expected hint inclusion for dangerous_command: %q", out2)
	}
}

func TestMapNetworkError(t *testing.T) {
	// Nil in, nil out
	if me := MapNetworkError("scan", nil); me != nil {
		t.Fatalf("expected nil for nil error")
	}
	// Permission mapping
	if me := MapNetworkError("scan", errors.New("permission denied: access")); me.Code != "permission_denied" {
		t.Fatalf("expected permission_denied, got %+v", me)
	}
	// Unsupported mapping
	if me := MapNetworkError("advertise", errors.New("not supported on this platform")); me.Code != "unsupported" {
		t.Fatalf("expected unsupported, got %+v", me)
	}
	// Adapter unavailable mapping
	if me := MapNetworkError("connect", errors.New("no adapter found")); me.Code != "adapter_unavailable" {
		t.Fatalf("expected adapter_unavailable, got %+v", me)
	}
	// Fallbacks by op
	if me := MapNetworkError("scan", errors.New("random failure")); me.Code != "scan_failed" {
		t.Fatalf("expected scan_failed, got %+v", me)
	}
	if me := MapNetworkError("advertise", errors.New("random failure")); me.Code != "advertise_failed" {
		t.Fatalf("expected advertise_failed, got %+v", me)
	}
	if me := MapNetworkError("connect", errors.New("random failure")); me.Code != "connect_failed" {
		t.Fatalf("expected connect_failed, got %+v", me)
	}
	if me := MapNetworkError("other", errors.New("random failure")); me.Code != "network_error" {
		t.Fatalf("expected network_error, got %+v", me)
	}
}

func TestIsMeshExecError(t *testing.T) {
	if IsMeshExecError(nil) {
		t.Fatalf("nil should not be MeshExecError")
	}
	if IsMeshExecError(errors.New("x")) {
		t.Fatalf("std error should not be MeshExecError")
	}
	if !IsMeshExecError(NewConfigError("invalid_config", "bad", nil)) {
		t.Fatalf("expected MeshExecError true")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(sub) > 0 && (indexOf(s, sub) >= 0)))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			if s[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
