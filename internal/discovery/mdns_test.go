package discovery

import (
	"context"
	"testing"
	"time"
)

func TestSplitKV(t *testing.T) {
	if kv := splitKV("a=b"); kv == nil || kv[0] != "a" || kv[1] != "b" {
		t.Fatalf("splitKV failed: %v", kv)
	}
	if kv := splitKV("noval"); kv != nil {
		t.Fatalf("expected nil for invalid input: %v", kv)
	}
}

func TestDiscover_ContextCancel(t *testing.T) {
	// Ensure function returns promptly when context times out
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, _ = Discover(ctx, 10*time.Millisecond)
}
