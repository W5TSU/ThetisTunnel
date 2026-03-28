package main

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"
)

// deadline wraps t.Helper and sets a read deadline, failing the test on error.
func deadline(t *testing.T, c interface{ SetReadDeadline(time.Time) error }, d time.Duration) {
	t.Helper()
	if err := c.SetReadDeadline(time.Now().Add(d)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
}

// freeUDPPort binds a UDP socket, records the port, closes it, and returns the port.
// There is a small TOCTOU window but it is acceptable for tests.
func freeUDPPort(t *testing.T) int {
	t.Helper()
	c, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freeUDPPort: %v", err)
	}
	port := c.LocalAddr().(*net.UDPAddr).Port
	c.Close()
	return port
}

// TestListenForwardsUDPToRadio verifies the TCP→UDP direction of the listen side:
// a frame arriving from the connect side is forwarded as UDP to the radio.
func TestListenForwardsUDPToRadio(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// "Radio" UDP socket — its port becomes the frame destination port.
	radioConn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("radio UDP listen: %v", err)
	}
	defer radioConn.Close()
	radioPort := uint16(radioConn.LocalAddr().(*net.UDPAddr).Port)

	// Proxy port that handleListenConn will bind.
	proxyPort := freeUDPPort(t)

	radioSrcPorts, _ := parsePorts(fmt.Sprintf("%d", radioPort))
	listenCfg := &Config{
		RadioIP:        "127.0.0.1",
		RadioProxyPort: proxyPort,
		RadioSrcRange:  strconv.Itoa(int(radioPort)),
		UDPBindIP:      "127.0.0.1",
		Key:            "testkey",
	}

	// In-process TCP pipe stands in for the real TCP connection.
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	// Start listen handler (radio site).
	go handleListenConn(ctx, serverConn, listenCfg, radioSrcPorts, &Stats{})

	// Simulate the connect side: authenticate then send a frame.
	if err := authConnect(clientConn, "testkey"); err != nil {
		t.Fatalf("authConnect: %v", err)
	}

	payload := []byte("hello from thetis")
	if err := writeFrame(clientConn, radioPort, payload); err != nil {
		t.Fatalf("writeFrame: %v", err)
	}

	// Radio should receive the UDP packet.
	deadline(t, radioConn, 3*time.Second)
	buf := make([]byte, 1024)
	n, _, err := radioConn.ReadFrom(buf)
	if err != nil {
		t.Fatalf("radio ReadFrom: %v", err)
	}
	if string(buf[:n]) != string(payload) {
		t.Errorf("radio received %q, want %q", buf[:n], payload)
	}
}

// TestListenForwardsUDPFromRadio verifies the UDP→TCP direction of the listen side:
// a UDP packet received from the radio is wrapped in a frame and sent to the connect side.
func TestListenForwardsUDPFromRadio(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Choose a proxy port for the listen side to bind on.
	proxyPort := freeUDPPort(t)

	// The "radio" will send from a known source port we can put in radioSrcRange.
	// Use an ephemeral port then record it.
	radioSendConn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("radio send socket: %v", err)
	}
	defer radioSendConn.Close()
	radioPort := uint16(radioSendConn.LocalAddr().(*net.UDPAddr).Port)

	radioSrcPorts, _ := parsePorts(strconv.Itoa(int(radioPort)))
	listenCfg := &Config{
		RadioIP:        "127.0.0.1",
		RadioProxyPort: proxyPort,
		RadioSrcRange:  strconv.Itoa(int(radioPort)),
		UDPBindIP:      "127.0.0.1",
		Key:            "",
	}

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	go handleListenConn(ctx, serverConn, listenCfg, radioSrcPorts, &Stats{})

	// Authenticate (empty key).
	if err := authConnect(clientConn, ""); err != nil {
		t.Fatalf("authConnect: %v", err)
	}

	// Give handleListenConn time to bind the proxy socket.
	time.Sleep(50 * time.Millisecond)

	// Radio sends a UDP packet to the proxy port.
	proxyAddr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("127.0.0.1:%d", proxyPort))
	reply := []byte("reply from radio")
	if _, err := radioSendConn.WriteTo(reply, proxyAddr); err != nil {
		t.Fatalf("radio WriteTo proxy: %v", err)
	}

	// The connect side (clientConn) should receive a TCP frame.
	clientConn.SetDeadline(time.Now().Add(3 * time.Second))
	port, payload, err := readFrame(clientConn)
	if err != nil {
		t.Fatalf("readFrame from listen side: %v", err)
	}
	if port != radioPort {
		t.Errorf("frame port = %d, want %d", port, radioPort)
	}
	if string(payload) != string(reply) {
		t.Errorf("frame payload = %q, want %q", payload, reply)
	}
}

// TestConnectForwardsUDPToTCP verifies that when Thetis sends a UDP packet to
// the connect instance it is wrapped in a TCP frame and forwarded to the listen side.
func TestConnectForwardsUDPToTCP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find a free port for the connect instance to intercept Thetis traffic on.
	thetisPort := freeUDPPort(t)

	// In-process TCP pipe stands in for the real TCP connection.
	connectConn, listenConn := net.Pipe()
	defer connectConn.Close()
	defer listenConn.Close()

	connectCfg := &Config{
		UDPBindIP:   "127.0.0.1",
		ThetisPorts: strconv.Itoa(thetisPort),
		Key:         "",
	}

	// runConnectOnConn assumes auth is already complete; start it directly.
	go runConnectOnConn(ctx, connectConn, connectCfg, &Stats{})

	// Give runConnectOnConn time to bind the UDP socket.
	time.Sleep(50 * time.Millisecond)

	// "Thetis" sends a UDP packet to the connect instance.
	thetisConn, err := net.Dial("udp4", fmt.Sprintf("127.0.0.1:%d", thetisPort))
	if err != nil {
		t.Fatalf("thetis dial: %v", err)
	}
	defer thetisConn.Close()

	payload := []byte("command from thetis")
	if _, err := thetisConn.Write(payload); err != nil {
		t.Fatalf("thetis write: %v", err)
	}

	// The listen side (listenConn) should receive a TCP frame.
	listenConn.SetDeadline(time.Now().Add(3 * time.Second))
	port, got, err := readFrame(listenConn)
	if err != nil {
		t.Fatalf("readFrame from connect side: %v", err)
	}
	if port != uint16(thetisPort) {
		t.Errorf("frame port = %d, want %d", port, thetisPort)
	}
	if string(got) != string(payload) {
		t.Errorf("frame payload = %q, want %q", got, payload)
	}
}

// TestConnectDeliversFrameToThetis verifies that a TCP frame received from the
// listen side is delivered as a UDP packet back to Thetis.
func TestConnectDeliversFrameToThetis(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	thetisPort := freeUDPPort(t)

	connectConn, listenConn := net.Pipe()
	defer connectConn.Close()
	defer listenConn.Close()

	connectCfg := &Config{
		UDPBindIP:   "127.0.0.1",
		ThetisPorts: strconv.Itoa(thetisPort),
		Key:         "",
	}

	go runConnectOnConn(ctx, connectConn, connectCfg, &Stats{})

	time.Sleep(50 * time.Millisecond)

	// Thetis sends first so connect learns the peer address.
	thetisConn, err := net.Dial("udp4", fmt.Sprintf("127.0.0.1:%d", thetisPort))
	if err != nil {
		t.Fatalf("thetis dial: %v", err)
	}
	defer thetisConn.Close()

	if _, err := thetisConn.Write([]byte("initial")); err != nil {
		t.Fatalf("thetis initial write: %v", err)
	}

	// Consume the forwarded frame so the TCP connection stays unblocked.
	listenConn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := readFrame(listenConn); err != nil {
		t.Fatalf("drain frame: %v", err)
	}
	listenConn.SetDeadline(time.Time{})

	// Now the listen side sends a reply frame back.
	reply := []byte("iq data from radio")
	if err := writeFrame(listenConn, uint16(thetisPort), reply); err != nil {
		t.Fatalf("writeFrame reply: %v", err)
	}

	// Thetis should receive the reply UDP packet.
	thetisConn.SetDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 1024)
	n, err := thetisConn.Read(buf)
	if err != nil {
		t.Fatalf("thetis Read: %v", err)
	}
	if string(buf[:n]) != string(reply) {
		t.Errorf("thetis received %q, want %q", buf[:n], reply)
	}
}
