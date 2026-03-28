package main

import "testing"

func TestParseRate(t *testing.T) {
	tests := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{"10M", 10_000_000, false},
		{"1G", 1_000_000_000, false},
		{"500K", 500_000, false},
		{"2214", 2214, false},
		{"10m", 10_000_000, false}, // lowercase suffix
		{"1g", 1_000_000_000, false},
		{"500k", 500_000, false},
		{"0", 0, false},
		{"", 0, false},
		{"abc", 0, true},
		{"10X", 0, true}, // unknown suffix treated as number
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseRate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseRate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseRate(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
