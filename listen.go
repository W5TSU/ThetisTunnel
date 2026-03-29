package main

// Listen mode — radio site
//
// Topology:
//
//   [connect instance] ──TCP──► [listen instance] ──UDP──► [radio]
//                                    radioProxyPort            radioIp:port
//
// All radio UDP traffic is funnelled through a single proxy socket (radioProxyPort).
// The radio's source port is carried in each TCP frame so the connect side knows
// which port to deliver the reply to.

import (
	"context"
	"fmt"
	"net"
)

func runListen(ctx context.Context, cfg *Config, stats *Stats, ulog *UILogger) error {
	if cfg.RadioIP == "" {
		return fmt.Errorf("--radioIp is required in listen mode")
	}
	if cfg.RadioProxyPort == 0 {
		return fmt.Errorf("--radioProxyPort is required in listen mode")
	}

	radioSrcPorts, err := parsePorts(cfg.RadioSrcRange)
	if err != nil {
		return fmt.Errorf("--radioSrcRange: %w", err)
	}

	tcpAddr := fmt.Sprintf("%s:%d", cfg.TCPBindIP, cfg.TCPPort)
	ln, err := net.Listen("tcp4", tcpAddr)
	if err != nil {
		return fmt.Errorf("TCP listen on %s: %w", tcpAddr, err)
	}
	defer ln.Close()

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	ulog.Log("Listening on TCP %s", tcpAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return fmt.Errorf("TCP accept: %w", err)
			}
		}
		ulog.Log("Connected: %s", conn.RemoteAddr())
		handleListenConn(ctx, conn, cfg, radioSrcPorts, stats, ulog)
		ulog.Log("Disconnected")
	}
}

func handleListenConn(ctx context.Context, tcpConn net.Conn, cfg *Config, radioSrcPorts []uint16, stats *Stats, ulog *UILogger) {
	defer tcpConn.Close()

	ulog.Debugf("Auth: reading key from %s", tcpConn.RemoteAddr())
	if err := authListen(tcpConn, cfg.Key); err != nil {
		ulog.Log("Auth failed from %s: %v", tcpConn.RemoteAddr(), err)
		return
	}
	ulog.Debugf("Auth: accepted key from %s", tcpConn.RemoteAddr())

	// Open the UDP proxy socket with SO_BROADCAST so discovery frames can be relayed.
	udpBindAddr := fmt.Sprintf("%s:%d", cfg.UDPBindIP, cfg.RadioProxyPort)
	ulog.Debugf("UDP: binding proxy socket on %s", udpBindAddr)
	udpConn, err := listenPacketBroadcast(ctx, udpBindAddr)
	if err != nil {
		ulog.Log("UDP bind on %s: %v", udpBindAddr, err)
		return
	}
	defer udpConn.Close()

	radioIP := net.ParseIP(cfg.RadioIP).To4()
	if radioIP == nil {
		ulog.Log("Invalid radio IP: %s", cfg.RadioIP)
		return
	}
	ulog.Debugf("Radio target: %s, src port filter: %v", cfg.RadioIP, cfg.RadioSrcRange)

	tcpReadDone := make(chan struct{})

	// TCP → UDP: frames arriving from the connect instance are forwarded to the radio.
	go func() {
		defer close(tcpReadDone)
		for {
			port, payload, err := readFrame(tcpConn)
			if err != nil {
				return
			}
			stats.TCPBytesIn.Add(int64(4 + len(payload)))

			dest := &net.UDPAddr{IP: radioIP, Port: int(port)}
			ulog.Debugf("TCP→UDP: port=%d len=%d → %s", port, len(payload), dest)
			n, err := udpConn.WriteTo(payload, dest)
			if err != nil {
				ulog.Log("UDP write to radio: %v", err)
				continue
			}
			stats.UDPBytesOut.Add(int64(n))
		}
	}()

	// UDP → TCP: replies from the radio are wrapped in frames and sent to the connect instance.
	go func() {
		buf := make([]byte, 65536)
		for {
			n, addr, err := udpConn.ReadFrom(buf)
			if err != nil {
				return
			}
			udpAddr, ok := addr.(*net.UDPAddr)
			if !ok {
				continue
			}
			srcPort := uint16(udpAddr.Port)
			if !portInList(radioSrcPorts, srcPort) {
				ulog.Debugf("UDP→TCP: dropped packet from %s (port %d not in src range)", addr, srcPort)
				continue
			}
			stats.UDPBytesIn.Add(int64(n))

			payload := make([]byte, n)
			copy(payload, buf[:n])

			if cfg.Broadcast1024 && srcPort == 1024 && isDiscoveryFrame(payload) {
				ulog.Debugf("UDP→TCP: discovery frame from port 1024, broadcasting on radio subnet")
				broadcastDiscovery(udpConn, payload, radioIP)
			}

			ulog.Debugf("UDP→TCP: port=%d len=%d from %s", srcPort, n, addr)
			if err := writeFrame(tcpConn, srcPort, payload); err != nil {
				return
			}
			stats.TCPBytesOut.Add(int64(4 + n))
		}
	}()

	select {
	case <-tcpReadDone:
	case <-ctx.Done():
	}
}

// isDiscoveryFrame checks for the OpenHPSDR discovery frame signature (0xEF 0xFE header).
func isDiscoveryFrame(payload []byte) bool {
	return len(payload) >= 2 && payload[0] == 0xEF && payload[1] == 0xFE
}

// broadcastDiscovery sends a discovery frame to the radio subnet's directed broadcast
// address and to 255.255.255.255.  A /24 subnet is assumed for the directed broadcast.
func broadcastDiscovery(conn net.PacketConn, payload []byte, radioIP net.IP) {
	directed := net.IP{radioIP[0], radioIP[1], radioIP[2], 255}
	for _, addr := range []string{
		fmt.Sprintf("%s:1024", directed),
		"255.255.255.255:1024",
	} {
		if dst, err := net.ResolveUDPAddr("udp4", addr); err == nil {
			conn.WriteTo(payload, dst) //nolint:errcheck
		}
	}
}
