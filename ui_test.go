package main

import (
	"sync/atomic"
	"testing"
)

func TestFormatBits(t *testing.T) {
	tests := []struct {
		bps  int64
		want string
	}{
		{0, "0 bps"},
		{999, "999 bps"},
		{1000, "1.00 Kbps"},
		{1500, "1.50 Kbps"},
		{1_000_000, "1.00 Mbps"},
		{10_000_000, "10.00 Mbps"},
		{1_000_000_000, "1.00 Gbps"},
		{2_500_000_000, "2.50 Gbps"},
	}
	for _, tt := range tests {
		got := formatBits(tt.bps)
		if got != tt.want {
			t.Errorf("formatBits(%d) = %q, want %q", tt.bps, got, tt.want)
		}
	}
}

func TestBarFrac(t *testing.T) {
	tests := []struct {
		value, maxValue int64
		width           int
		want            int
	}{
		{0, 100, 30, 0},
		{50, 100, 30, 15},
		{100, 100, 30, 30},
		{200, 100, 30, 30},  // clamped to width
		{0, 0, 30, 0},       // zero max
		{10, 0, 30, 0},      // zero max
		{-1, 100, 30, 0},    // negative value
	}
	for _, tt := range tests {
		got := barFrac(tt.value, tt.maxValue, tt.width)
		if got != tt.want {
			t.Errorf("barFrac(%d, %d, %d) = %d, want %d", tt.value, tt.maxValue, tt.width, got, tt.want)
		}
	}
}

func TestUpdatePeak(t *testing.T) {
	var p atomic.Int64

	updatePeak(&p, 100)
	if p.Load() != 100 {
		t.Errorf("expected 100, got %d", p.Load())
	}

	updatePeak(&p, 50) // lower value — should not change peak
	if p.Load() != 100 {
		t.Errorf("peak should remain 100, got %d", p.Load())
	}

	updatePeak(&p, 200) // new high
	if p.Load() != 200 {
		t.Errorf("expected 200, got %d", p.Load())
	}
}

func TestStatsResetPeaks(t *testing.T) {
	s := &Stats{}
	s.PeakTCPIn.Store(1000)
	s.PeakTCPOut.Store(2000)
	s.PeakUDPIn.Store(3000)
	s.PeakUDPOut.Store(4000)

	s.ResetPeaks()

	for name, val := range map[string]int64{
		"PeakTCPIn":  s.PeakTCPIn.Load(),
		"PeakTCPOut": s.PeakTCPOut.Load(),
		"PeakUDPIn":  s.PeakUDPIn.Load(),
		"PeakUDPOut": s.PeakUDPOut.Load(),
	} {
		if val != 0 {
			t.Errorf("%s after ResetPeaks = %d, want 0", name, val)
		}
	}
}
