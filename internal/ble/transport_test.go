package ble

import (
    "context"
    "testing"
    "time"
)

func TestTransport_ScanReceivesAdvertisement(t *testing.T) {
    tr := NewTransportWithLogger(nil)
    ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
    defer cancel()

    advCh, err := tr.Scan(ctx)
    if err != nil {
        t.Fatalf("scan error: %v", err)
    }

    advCtx, advCancel := context.WithCancel(context.Background())
    defer advCancel()
    if err := tr.Advertise(advCtx, []byte("data")); err != nil {
        t.Fatalf("advertise error: %v", err)
    }

    select {
    case <-ctx.Done():
        t.Fatal("timeout waiting for advertisement")
    case adv, ok := <-advCh:
        if !ok || adv == nil {
            t.Fatal("advertisement channel closed unexpectedly")
        }
    }
}

func TestTransport_Connect_InvalidMAC(t *testing.T) {
    tr := NewTransportWithLogger(nil)
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()
    if _, err := tr.Connect(ctx, "not-a-mac"); err == nil {
        t.Fatal("expected error for invalid MAC address")
    }
}

func TestTransport_CreateGATTService(t *testing.T) {
    tr := NewTransportWithLogger(nil)
    svc, err := tr.CreateGATTService()
    if err != nil {
        t.Fatalf("CreateGATTService error: %v", err)
    }
    if svc == nil || svc.UUID == "" || len(svc.Characteristics) == 0 {
        t.Fatalf("unexpected service: %+v", svc)
    }
}


