package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

// Config holds all runtime configuration parsed from CLI flags.
type Config struct {
	Mode           string // "listen" or "connect"
	TCPPort        int
	TCPBindIP      string
	TCPHost        string
	UDPBindIP      string
	RadioIP        string
	RadioProxyPort int
	RadioSrcRange  string
	Broadcast1024  bool
	ThetisPorts    string
	Key            string
	NoUI           bool
	MaxInRate      int64 // bits/sec, for UI bar scaling
	MaxOutRate     int64
}

func main() {
	mode := flag.String("mode", "", "listen or connect")
	tcpPort := flag.Int("tcpPort", 0, "TCP port for the tunnel")
	tcpBindIP := flag.String("tcpBindIP", "0.0.0.0", "Local IP to bind TCP listener (listen mode)")
	tcpHost := flag.String("tcpHost", "", "Listener host/IP or hostname (connect mode)")
	udpBindIP := flag.String("udpBindIP", "0.0.0.0", "Local IP to bind UDP sockets")
	radioIP := flag.String("radioIp", "", "Radio IPv4 address (listen mode)")
	radioProxyPort := flag.Int("radioProxyPort", 0, "Local UDP port for radio proxy (listen mode)")
	radioSrcRange := flag.String("radioSrcRange", "1024-1042", "Radio source UDP port range (listen mode)")
	broadcast1024 := flag.Bool("broadcast1024", false, "Broadcast port 1024 discovery frames (listen mode)")
	thetisPorts := flag.String("thetisPorts", "1024-1029", "Thetis UDP destination ports (connect mode)")
	key := flag.String("key", "", "Shared secret key, max 32 chars")
	noUI := flag.Bool("noUI", false, "Disable live throughput display")
	maxInRate := flag.String("maxInRate", "10M", "Max IN rate for display bar scaling (e.g. 10M, 1G)")
	maxOutRate := flag.String("maxOutRate", "10M", "Max OUT rate for display bar scaling (e.g. 10M, 1G)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `A TCP tunnel for Thetis UDP traffic.
Run one instance at the radio site (listen) and one at the Thetis site (connect).

Usage:
  thetistunnel --mode=listen  --tcpPort=PORT --tcpBindIP=IP --udpBindIP=IP --radioIp=IP --radioProxyPort=PORT [options]
  thetistunnel --mode=connect --tcpHost=HOST --tcpPort=PORT --udpBindIP=IP [--thetisPorts=RANGE] [options]

Options:
`)
		flag.PrintDefaults()
	}
	flag.Parse()

	if *mode != "listen" && *mode != "connect" {
		fmt.Fprintln(os.Stderr, "error: --mode must be 'listen' or 'connect'")
		flag.Usage()
		os.Exit(1)
	}
	if *tcpPort == 0 {
		fmt.Fprintln(os.Stderr, "error: --tcpPort is required")
		os.Exit(1)
	}
	if len(*key) > 32 {
		fmt.Fprintln(os.Stderr, "error: --key must not exceed 32 characters")
		os.Exit(1)
	}

	inRate, err := parseRate(*maxInRate)
	if err != nil {
		log.Fatalf("--maxInRate: %v", err)
	}
	outRate, err := parseRate(*maxOutRate)
	if err != nil {
		log.Fatalf("--maxOutRate: %v", err)
	}

	cfg := &Config{
		Mode:           *mode,
		TCPPort:        *tcpPort,
		TCPBindIP:      *tcpBindIP,
		TCPHost:        *tcpHost,
		UDPBindIP:      *udpBindIP,
		RadioIP:        *radioIP,
		RadioProxyPort: *radioProxyPort,
		RadioSrcRange:  *radioSrcRange,
		Broadcast1024:  *broadcast1024,
		ThetisPorts:    *thetisPorts,
		Key:            *key,
		NoUI:           *noUI,
		MaxInRate:      inRate,
		MaxOutRate:     outRate,
	}

	stats := &Stats{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		switch cfg.Mode {
		case "listen":
			errCh <- runListen(ctx, cfg, stats)
		case "connect":
			errCh <- runConnect(ctx, cfg, stats)
		}
	}()

	if !*noUI {
		go runUI(ctx, stats, cfg)
	}
	go handleKeyboard(cancel, stats)

	select {
	case sig := <-sigCh:
		fmt.Printf("\nReceived %v, shutting down.\n", sig)
	case err := <-errCh:
		if err != nil {
			log.Fatalf("fatal: %v", err)
		}
	case <-ctx.Done():
	}
}

// parseRate converts a human-readable rate string (e.g. "10M", "1G", "500K", "2214")
// into bits per second.
func parseRate(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	mult := int64(1)
	switch strings.ToUpper(string(s[len(s)-1])) {
	case "G":
		mult = 1_000_000_000
		s = s[:len(s)-1]
	case "M":
		mult = 1_000_000
		s = s[:len(s)-1]
	case "K":
		mult = 1_000
		s = s[:len(s)-1]
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid rate %q", s)
	}
	return v * mult, nil
}
