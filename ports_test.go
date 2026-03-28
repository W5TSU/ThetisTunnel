package main

import (
	"reflect"
	"testing"
)

func TestParsePorts(t *testing.T) {
	tests := []struct {
		input   string
		want    []uint16
		wantErr bool
	}{
		{"1024-1029", []uint16{1024, 1025, 1026, 1027, 1028, 1029}, false},
		{"1024", []uint16{1024}, false},
		{"1024,1025", []uint16{1024, 1025}, false},
		{"1024,1030-1032", []uint16{1024, 1030, 1031, 1032}, false},
		{"1024-1042", makeRange(1024, 1042), false},
		{"5000-5000", []uint16{5000}, false},
		{"1025-1024", nil, true},  // inverted range
		{"abc", nil, true},        // non-numeric
		{"99999", nil, true},      // overflow uint16
		{"1024-abc", nil, true},   // bad high end
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parsePorts(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parsePorts(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePorts(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPortInList(t *testing.T) {
	ports := []uint16{1024, 1025, 1030}
	if !portInList(ports, 1024) {
		t.Error("expected 1024 to be in list")
	}
	if !portInList(ports, 1030) {
		t.Error("expected 1030 to be in list")
	}
	if portInList(ports, 1026) {
		t.Error("expected 1026 not to be in list")
	}
	if portInList(nil, 1024) {
		t.Error("expected false for nil list")
	}
}

func makeRange(lo, hi uint16) []uint16 {
	out := make([]uint16, 0, hi-lo+1)
	for p := lo; p <= hi; p++ {
		out = append(out, p)
	}
	return out
}
