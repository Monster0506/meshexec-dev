package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/grandcat/zeroconf"
)

// fakeResolver implements mdnsResolver without network access
type fakeResolver struct{}

var _ mdnsResolver = (*fakeResolver)(nil)

func (f *fakeResolver) Browse(ctx context.Context, service, domain string, entries chan<- *zeroconf.ServiceEntry) error {
	// Do not send entries; just return
	return nil
}

func TestSplitKV(t *testing.T) {
	if kv := splitKV("a=b"); kv == nil || kv[0] != "a" || kv[1] != "b" {
		t.Fatalf("splitKV failed: %v", kv)
	}
	if kv := splitKV("noval"); kv != nil {
		t.Fatalf("expected nil for invalid input: %v", kv)
	}
}

func TestDiscover_ContextCancel(t *testing.T) {
	// Stub resolver to avoid network
	old := newResolver
	defer func() { newResolver = old }()
	newResolver = func() (mdnsResolver, error) { return &fakeResolver{}, nil }

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, _ = Discover(ctx, 10*time.Millisecond)
}
