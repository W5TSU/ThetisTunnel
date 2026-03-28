package main

import "testing"

func TestIsDiscoveryFrame(t *testing.T) {
	tests := []struct {
		payload []byte
		want    bool
	}{
		{[]byte{0xEF, 0xFE, 0x01, 0x02}, true},
		{[]byte{0xEF, 0xFE}, true},
		{[]byte{0xEF, 0xFF}, false},        // wrong second byte
		{[]byte{0x00, 0xFE}, false},         // wrong first byte
		{[]byte{0xEF}, false},               // too short
		{[]byte{}, false},                   // empty
	}
	for _, tt := range tests {
		got := isDiscoveryFrame(tt.payload)
		if got != tt.want {
			t.Errorf("isDiscoveryFrame(%#v) = %v, want %v", tt.payload, got, tt.want)
		}
	}
}
