package main

import (
	"net"
	"testing"
)

// pipe returns a connected pair of net.Conn backed by an in-process pipe.
func pipe(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	a, b := net.Pipe()
	t.Cleanup(func() { a.Close(); b.Close() })
	return a, b
}

func TestFrameRoundTrip(t *testing.T) {
	a, b := pipe(t)

	want := []byte("hello radio")
	wantPort := uint16(1024)

	errc := make(chan error, 1)
	go func() {
		errc <- writeFrame(a, wantPort, want)
	}()

	gotPort, gotPayload, err := readFrame(b)
	if err != nil {
		t.Fatalf("readFrame: %v", err)
	}
	if err := <-errc; err != nil {
		t.Fatalf("writeFrame: %v", err)
	}
	if gotPort != wantPort {
		t.Errorf("port = %d, want %d", gotPort, wantPort)
	}
	if string(gotPayload) != string(want) {
		t.Errorf("payload = %q, want %q", gotPayload, want)
	}
}

func TestFrameMultiple(t *testing.T) {
	a, b := pipe(t)

	frames := []struct {
		port    uint16
		payload string
	}{
		{1024, "discovery"},
		{1025, "iq data"},
		{1030, "mic audio"},
	}

	go func() {
		for _, f := range frames {
			if err := writeFrame(a, f.port, []byte(f.payload)); err != nil {
				return
			}
		}
	}()

	for _, want := range frames {
		port, payload, err := readFrame(b)
		if err != nil {
			t.Fatalf("readFrame: %v", err)
		}
		if port != want.port {
			t.Errorf("port = %d, want %d", port, want.port)
		}
		if string(payload) != want.payload {
			t.Errorf("payload = %q, want %q", payload, want.payload)
		}
	}
}

func TestFrameEmptyPayload(t *testing.T) {
	a, b := pipe(t)

	go writeFrame(a, 1024, []byte{}) //nolint:errcheck

	port, payload, err := readFrame(b)
	if err != nil {
		t.Fatalf("readFrame: %v", err)
	}
	if port != 1024 {
		t.Errorf("port = %d, want 1024", port)
	}
	if len(payload) != 0 {
		t.Errorf("expected empty payload, got %v", payload)
	}
}

func TestAuthAccepted(t *testing.T) {
	a, b := pipe(t)

	errc := make(chan error, 1)
	go func() { errc <- authListen(a, "secret") }()

	if err := authConnect(b, "secret"); err != nil {
		t.Fatalf("authConnect: %v", err)
	}
	if err := <-errc; err != nil {
		t.Fatalf("authListen: %v", err)
	}
}

func TestAuthRejected(t *testing.T) {
	a, b := pipe(t)

	go authListen(a, "correct") //nolint:errcheck

	err := authConnect(b, "wrong")
	if err == nil {
		t.Fatal("expected auth to be rejected, got nil error")
	}
}

func TestAuthNoKey(t *testing.T) {
	// Listener with no key should accept any key from connector.
	a, b := pipe(t)

	errc := make(chan error, 1)
	go func() { errc <- authListen(a, "") }()

	if err := authConnect(b, "anything"); err != nil {
		t.Fatalf("authConnect: %v", err)
	}
	if err := <-errc; err != nil {
		t.Fatalf("authListen: %v", err)
	}
}

func TestAuthKeyPadding(t *testing.T) {
	// Keys shorter than 32 bytes must still match when padded with zeros.
	a, b := pipe(t)

	errc := make(chan error, 1)
	go func() { errc <- authListen(a, "short") }()

	if err := authConnect(b, "short"); err != nil {
		t.Fatalf("authConnect: %v", err)
	}
	if err := <-errc; err != nil {
		t.Fatalf("authListen: %v", err)
	}
}
