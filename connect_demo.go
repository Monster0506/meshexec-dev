package main

import (
  "context"
  "flag"
  "fmt"
  "time"

  "github.com/monster0506/mechexec/internal"
  "github.com/monster0506/mechexec/internal/ble"
)

func main() {
  addrFlag := flag.String("addr", "", "BLE device address")
  flag.Parse()

  cfg := internal.DefaultConfig()
  t, err := ble.New(&cfg.Network)
  if err != nil { panic(err) }

  // Start advertising so the Windows simulator can discover itself
  advCtx, advCancel := context.WithCancel(context.Background())
  defer advCancel()
  _ = t.Advertise(advCtx, []byte("mechexec"))

  m := ble.NewManager(t)

  ctx, cancel := context.WithCancel(context.Background())
  defer cancel()
  if err := m.StartDiscovery(ctx); err != nil { panic(err) }

  target := *addrFlag
  if target == "" {
    subCtx, subCancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer subCancel()
    updates := m.Subscribe(subCtx)
    select {
    case p := <-updates:
      target = p.Address
      fmt.Println("Discovered:", p.Name, p.Address)
    case <-subCtx.Done():
      panic("no device discovered")
    }
  }

  conn, err := m.Connect(ctx, target)
  if err != nil { panic(err) }
  fmt.Printf("Connected to %s (MTU=%d)\n", conn.Address, conn.MTU)
}
