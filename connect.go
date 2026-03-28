package main

// Connect mode — Thetis site
//
// Topology:
//
//   [Thetis] ──UDP──► [connect instance] ──TCP──► [listen instance]
//                         udpBindIP:port               radioProxyPort
//
// Connect binds one UDP socket per port in --thetisPorts.  Thetis is configured
// to treat the connect machine as the radio (e.g. point to 127.0.0.1 or the
// local LAN IP).  When a frame arrives back from the listen side the payload is
// delivered to Thetis using the same port number as the frame's port field —
// i.e. the radio's original source port — which is where Thetis listens for
// incoming data from the radio.
//
// Sender tracking: when Thetis sends UDP to port P the source address is
// remembered.  Replies from the listen side for port P are sent back to that
// address.  For ports that the radio sends to but Thetis has not yet sent on,
// the last-seen Thetis peer IP is used with the frame's port number as the
// destination port.

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
)

func runConnect(ctx context.Context, cfg *Config, stats *Stats) error {
	if cfg.TCPHost == "" {
		return fmt.Errorf("--tcpHost is required in connect mode")
	}

	tcpAddr := fmt.Sprintf("%s:%d", cfg.TCPHost, cfg.TCPPort)
	fmt.Printf("Connecting to TCP %s\n", tcpAddr)

	tcpConn, err := net.Dial("tcp4", tcpAddr)
	if err != nil {
		return fmt.Errorf("TCP connect to %s: %w", tcpAddr, err)
	}
	defer tcpConn.Close()

	go func() {
		<-ctx.Done()
		tcpConn.Close()
	}()

	if err := authConnect(tcpConn, cfg.Key); err != nil {
		return fmt.Errorf("authentication: %w", err)
	}
	fmt.Println("Connected and authenticated.")

	return runConnectOnConn(ctx, tcpConn, cfg, stats)
}

// runConnectOnConn is the inner loop for connect mode.  It expects the TCP
// connection to be already authenticated and takes ownership of it.
// Extracted as a separate function so tests can inject an in-process pipe.
func runConnectOnConn(ctx context.Context, tcpConn net.Conn, cfg *Config, stats *Stats) error {
	thetisPortList, err := parsePorts(cfg.ThetisPorts)
	if err != nil {
		return fmt.Errorf("--thetisPorts: %w", err)
	}

	// Bind one UDP socket per Thetis port.
	udpSockets := make(map[uint16]net.PacketConn, len(thetisPortList))
	for _, port := range thetisPortList {
		addr := fmt.Sprintf("%s:%d", cfg.UDPBindIP, port)
		pc, err := net.ListenPacket("udp4", addr)
		if err != nil {
			for _, s := range udpSockets {
				s.Close()
			}
			return fmt.Errorf("UDP bind on %s: %w", addr, err)
		}
		udpSockets[port] = pc
		fmt.Printf("Intercepting Thetis UDP on %s\n", addr)
	}
	defer func() {
		for _, s := range udpSockets {
			s.Close()
		}
	}()

	// thetisPeerIP holds the most-recently-seen Thetis sender IP.
	var thetisPeerIP atomic.Pointer[net.IP]

	// lastSender maps bind-port → last Thetis sender address.
	var mu sync.RWMutex
	lastSender := make(map[uint16]*net.UDPAddr, len(thetisPortList))

	tcpReadDone := make(chan struct{})

	// Per-port goroutines: Thetis UDP → TCP frames.
	for _, port := range thetisPortList {
		port := port
		pc := udpSockets[port]
		go func() {
			buf := make([]byte, 65536)
			for {
				n, addr, err := pc.ReadFrom(buf)
				if err != nil {
					select {
					case <-tcpReadDone:
					default:
						log.Printf("UDP read on port %d: %v", port, err)
					}
					return
				}
				udpAddr, ok := addr.(*net.UDPAddr)
				if !ok {
					continue
				}

				mu.Lock()
				lastSender[port] = udpAddr
				mu.Unlock()
				ip := udpAddr.IP
				thetisPeerIP.Store(&ip)

				stats.UDPBytesIn.Add(int64(n))

				payload := make([]byte, n)
				copy(payload, buf[:n])

				if err := writeFrame(tcpConn, port, payload); err != nil {
					return
				}
				stats.TCPBytesOut.Add(int64(4 + n))
			}
		}()
	}

	// TCP read goroutine: frames from listen side → Thetis UDP.
	go func() {
		defer close(tcpReadDone)
		for {
			port, payload, err := readFrame(tcpConn)
			if err != nil {
				return
			}
			stats.TCPBytesIn.Add(int64(4 + len(payload)))

			// Determine the destination address for the reply.
			mu.RLock()
			target := lastSender[port]
			mu.RUnlock()

			if target == nil {
				// No traffic seen on this exact port yet; use the last-known
				// Thetis IP with the frame's port as the destination port.
				if peerIP := thetisPeerIP.Load(); peerIP != nil {
					target = &net.UDPAddr{IP: *peerIP, Port: int(port)}
				}
			}
			if target == nil {
				continue // no Thetis address known yet; drop
			}

			// Prefer the socket bound on the matching port; fall back to any.
			pc, ok := udpSockets[port]
			if !ok {
				for _, s := range udpSockets {
					pc = s
					break
				}
			}
			if pc == nil {
				continue
			}

			n, err := pc.WriteTo(payload, target)
			if err != nil {
				log.Printf("UDP write to Thetis port %d: %v", port, err)
				continue
			}
			stats.UDPBytesOut.Add(int64(n))
		}
	}()

	select {
	case <-tcpReadDone:
	case <-ctx.Done():
	}
	return nil
}
