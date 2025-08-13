//go:build dev

package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	pocAddr string
	pocText string
)

// poc-listen starts a simple TCP server that prints any received text and echoes an ACK
var pocListenCmd = &cobra.Command{
	Use:   "poc-listen",
	Short: "Start a simple TCP listener for POC (no BLE)",
	Run: func(cmd *cobra.Command, args []string) {
		addr := pocAddr
		if addr == "" {
			addr = ":9876"
		}
		if logger != nil {
			logger.Info("poc-listen: starting", map[string]interface{}{"addr": addr})
		}
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "listen error: %v\n", err)
			os.Exit(1)
		}
		defer func() { _ = ln.Close() }()
		for {
			conn, err := ln.Accept()
			if err != nil {
				fmt.Fprintf(os.Stderr, "accept error: %v\n", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				r := bufio.NewReader(c)
				line, _ := r.ReadString('\n')
				if logger != nil {
					logger.Info("poc-listen: received", map[string]interface{}{"len": len(line)})
				}
				_, _ = io.WriteString(c, "ACK\n")
				fmt.Print(line)
			}(conn)
		}
	},
}

// poc-send connects to a TCP endpoint and sends text, printing any response
var pocSendCmd = &cobra.Command{
	Use:   "poc-send",
	Short: "Send a simple TCP message for POC (no BLE)",
	Run: func(cmd *cobra.Command, args []string) {
		addr := pocAddr
		if addr == "" {
			fmt.Fprintln(os.Stderr, "--addr required, e.g. 192.168.1.10:9876")
			os.Exit(2)
		}
		text := pocText
		if text == "" {
			text = "hello from meshexec POC"
		}
		if logger != nil {
			logger.Info("poc-send: connecting", map[string]interface{}{"addr": addr})
		}
		d := net.Dialer{Timeout: 5 * time.Second}
		conn, err := d.Dial("tcp", addr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dial error: %v\n", err)
			os.Exit(3)
		}
		defer func() { _ = conn.Close() }()
		if _, err := io.WriteString(conn, text+"\n"); err != nil {
			fmt.Fprintf(os.Stderr, "write error: %v\n", err)
			os.Exit(4)
		}
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		resp, _ := bufio.NewReader(conn).ReadString('\n')
		if resp != "" {
			fmt.Print(resp)
		}
		if logger != nil {
			logger.Info("poc-send: done", map[string]interface{}{"bytes": len(text)})
		}
	},
}

func init() {
	pocListenCmd.Flags().StringVar(&pocAddr, "addr", ":9876", "listen address (host:port)")
	rootCmd.AddCommand(pocListenCmd)

	pocSendCmd.Flags().StringVar(&pocAddr, "addr", "", "destination address (host:port)")
	pocSendCmd.Flags().StringVar(&pocText, "text", "", "text to send")
	rootCmd.AddCommand(pocSendCmd)
}
