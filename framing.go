package main

// Wire protocol
//
// Authentication handshake (always performed after TCP connect):
//   Connect → Listen : [32 bytes] key, null-padded to 32 bytes
//   Listen  → Connect: [1 byte]  0x01 = accepted, 0x00 = rejected
//
// If the listener has no key configured it accepts any key.
//
// Data frames:
//   [2 bytes BE] port number
//   [2 bytes BE] payload length
//   [N bytes]    payload (raw UDP datagram contents)

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

const authKeySize = 32

// authConnect sends the shared key to the listener and checks its response.
func authConnect(conn net.Conn, key string) error {
	var buf [authKeySize]byte
	copy(buf[:], key)
	if _, err := conn.Write(buf[:]); err != nil {
		return fmt.Errorf("sending auth key: %w", err)
	}
	var resp [1]byte
	if _, err := io.ReadFull(conn, resp[:]); err != nil {
		return fmt.Errorf("reading auth response: %w", err)
	}
	if resp[0] != 0x01 {
		return fmt.Errorf("authentication rejected by listener")
	}
	return nil
}

// authListen reads the key from the connector and validates it.
// If the listener's own key is empty, any key is accepted.
func authListen(conn net.Conn, key string) error {
	var received [authKeySize]byte
	if _, err := io.ReadFull(conn, received[:]); err != nil {
		return fmt.Errorf("reading auth key: %w", err)
	}
	if key != "" {
		var expected [authKeySize]byte
		copy(expected[:], key)
		if received != expected {
			conn.Write([]byte{0x00}) //nolint:errcheck
			return fmt.Errorf("key mismatch")
		}
	}
	_, err := conn.Write([]byte{0x01})
	return err
}

// writeFrame sends a framed UDP payload over the TCP connection.
func writeFrame(conn net.Conn, port uint16, payload []byte) error {
	buf := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint16(buf[0:2], port)
	binary.BigEndian.PutUint16(buf[2:4], uint16(len(payload)))
	copy(buf[4:], payload)
	_, err := conn.Write(buf)
	return err
}

// readFrame reads one framed UDP payload from the TCP connection.
func readFrame(conn net.Conn) (port uint16, payload []byte, err error) {
	var hdr [4]byte
	if _, err = io.ReadFull(conn, hdr[:]); err != nil {
		return
	}
	port = binary.BigEndian.Uint16(hdr[0:2])
	length := int(binary.BigEndian.Uint16(hdr[2:4]))
	payload = make([]byte, length)
	_, err = io.ReadFull(conn, payload)
	return
}
