package main

import "sync/atomic"

// Stats tracks cumulative byte counters for the live throughput display.
// All fields are updated concurrently from multiple goroutines.
type Stats struct {
	TCPBytesIn  atomic.Int64
	TCPBytesOut atomic.Int64
	UDPBytesIn  atomic.Int64
	UDPBytesOut atomic.Int64

	PeakTCPIn  atomic.Int64
	PeakTCPOut atomic.Int64
	PeakUDPIn  atomic.Int64
	PeakUDPOut atomic.Int64
}

func (s *Stats) ResetPeaks() {
	s.PeakTCPIn.Store(0)
	s.PeakTCPOut.Store(0)
	s.PeakUDPIn.Store(0)
	s.PeakUDPOut.Store(0)
}
