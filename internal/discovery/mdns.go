package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/grandcat/zeroconf"
	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

const serviceType = "_meshexec._tcp"

// package logger (default disabled); can be set via SetLogger
var pkgLogger *logging.Logger = logging.NewLogger("none")

// SetLogger sets discovery package logger
func SetLogger(l *logging.Logger) {
	if l != nil {
		pkgLogger = l
	}
}

// Advertiser wraps a zeroconf server
type Advertiser struct {
	server *zeroconf.Server
	logger *logging.Logger
}

// resolver seam for tests
type mdnsResolver interface {
	Browse(ctx context.Context, service, domain string, entries chan<- *zeroconf.ServiceEntry) error
}

var newResolver = func() (mdnsResolver, error) { return zeroconf.NewResolver(nil) }

// StartAdvertiser publishes the local node over mDNS with provided metadata
func StartAdvertiser(instance string, port int, meta map[string]string) (*Advertiser, error) {
	if port <= 0 {
		return nil, fmt.Errorf("invalid port")
	}
	var txt []string
	for k, v := range meta {
		txt = append(txt, fmt.Sprintf("%s=%s", k, v))
	}
	if pkgLogger != nil {
		pkgLogger.Debug("mdns: registering service", map[string]interface{}{"instance": instance, "port": port, "txt_len": len(txt)})
	}
	srv, err := zeroconf.Register(instance, serviceType, "local.", port, txt, nil)
	if err != nil {
		if pkgLogger != nil {
			pkgLogger.Warn("mdns: register failed", map[string]interface{}{"error": err.Error()})
		}
		return nil, err
	}
	if pkgLogger != nil {
		pkgLogger.Debug("mdns: advertiser started", map[string]interface{}{"instance": instance, "port": port, "meta_keys": len(meta)})
	}
	return &Advertiser{server: srv, logger: pkgLogger}, nil
}

// Stop stops the advertiser
func (a *Advertiser) Stop() {
	if a != nil && a.server != nil {
		if a.logger != nil {
			a.logger.Info("mdns: advertiser stopped", nil)
		}
		a.server.Shutdown()
	}
}

// Discover finds peers advertising the service within the timeout
func Discover(ctx context.Context, timeout time.Duration) ([]core.PeerInfo, error) {
	lg := pkgLogger
	if lg != nil {
		lg.Debug("mdns: discover begin", map[string]interface{}{"timeout_ms": timeout.Milliseconds(), "service": serviceType})
	}
	r, err := newResolver()
	if err != nil {
		if lg != nil {
			lg.Warn("mdns: resolver create failed", map[string]interface{}{"error": err.Error()})
		}
		return nil, err
	}
	entries := make(chan *zeroconf.ServiceEntry, 32)
	var out []core.PeerInfo
	go func() {
		for e := range entries {
			addr := firstAddr(e)
			meta := make(map[string]string)
			for _, t := range e.Text {
				if kv := splitKV(t); kv != nil {
					meta[kv[0]] = kv[1]
				}
			}
			if lg != nil {
				lg.Debug("mdns: entry", map[string]interface{}{"instance": e.Instance, "port": e.Port, "addr": addr, "txt": len(e.Text)})
			}
			out = append(out, core.PeerInfo{
				ID:       addr,
				Name:     e.Instance,
				Address:  fmt.Sprintf("%s:%d", addr, e.Port),
				Role:     meta["role"],
				OS:       meta["os"],
				Arch:     meta["arch"],
				Tags:     meta,
				LastSeen: time.Now(),
			})
		}
		if lg != nil {
			lg.Debug("mdns: entries channel closed", nil)
		}
	}()
	qctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if lg != nil {
		lg.Debug("mdns: browse start", nil)
	}
	if err := r.Browse(qctx, serviceType, "local.", entries); err != nil {
		if lg != nil {
			lg.Warn("mdns: browse failed", map[string]interface{}{"error": err.Error()})
		}
		return nil, err
	}
	<-qctx.Done()
	if lg != nil {
		lg.Info("mdns: discovery complete", map[string]interface{}{"count": len(out), "timeout_ms": timeout.Milliseconds()})
	}
	return out, nil
}

func firstAddr(e *zeroconf.ServiceEntry) string {
	if len(e.AddrIPv4) > 0 {
		return e.AddrIPv4[0].String()
	}
	if len(e.AddrIPv6) > 0 {
		return e.AddrIPv6[0].String()
	}
	return ""
}

func splitKV(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}
