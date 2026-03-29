package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

func runUI(ctx context.Context, stats *Stats, cfg *Config, ulog *UILogger) {
	var prevTCPIn, prevTCPOut, prevUDPIn, prevUDPOut int64

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// Clear screen, hide cursor.  In raw mode \n does not produce \r\n, so all
	// output in this function must use \r\n and \033[2K to avoid diagonal text.
	fmt.Print("\033[2J\033[H\033[?25l")
	defer fmt.Print("\033[?25h\r\n")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		curTCPIn := stats.TCPBytesIn.Load()
		curTCPOut := stats.TCPBytesOut.Load()
		curUDPIn := stats.UDPBytesIn.Load()
		curUDPOut := stats.UDPBytesOut.Load()

		// Rates in bits/sec (ticker fires every second).
		rateTCPIn := (curTCPIn - prevTCPIn) * 8
		rateTCPOut := (curTCPOut - prevTCPOut) * 8
		rateUDPIn := (curUDPIn - prevUDPIn) * 8
		rateUDPOut := (curUDPOut - prevUDPOut) * 8

		prevTCPIn = curTCPIn
		prevTCPOut = curTCPOut
		prevUDPIn = curUDPIn
		prevUDPOut = curUDPOut

		updatePeak(&stats.PeakTCPIn, rateTCPIn)
		updatePeak(&stats.PeakTCPOut, rateTCPOut)
		updatePeak(&stats.PeakUDPIn, rateUDPIn)
		updatePeak(&stats.PeakUDPOut, rateUDPOut)

		fmt.Print("\033[H") // move cursor to top-left
		fmt.Printf("\033[2KThetisTunnel  mode=%-7s  [Q=quit  R=reset peaks]\r\n", cfg.Mode)
		fmt.Print("\033[2K─────────────────────────────────────────────────────────────\r\n")
		printRate("TCP  IN ", rateTCPIn, stats.PeakTCPIn.Load(), cfg.MaxInRate)
		printRate("TCP  OUT", rateTCPOut, stats.PeakTCPOut.Load(), cfg.MaxOutRate)
		printRate("UDP  IN ", rateUDPIn, stats.PeakUDPIn.Load(), cfg.MaxInRate)
		printRate("UDP  OUT", rateUDPOut, stats.PeakUDPOut.Load(), cfg.MaxOutRate)
		fmt.Print("\033[2K\r\n")
		for _, line := range ulog.Lines() {
			fmt.Printf("\033[2K  %s\r\n", line)
		}
		fmt.Print("\033[J") // erase to end of screen
	}
}

// updatePeak atomically sets *p to max(*p, v).
func updatePeak(p *atomic.Int64, v int64) {
	for {
		cur := p.Load()
		if v <= cur {
			return
		}
		if p.CompareAndSwap(cur, v) {
			return
		}
	}
}

const barWidth = 32

// printRate renders one labelled throughput bar with a peak marker.
func printRate(label string, rate, peak, maxRate int64) {
	if maxRate <= 0 {
		maxRate = 10_000_000
	}

	filled := barFrac(rate, maxRate, barWidth)
	peakPos := barFrac(peak, maxRate, barWidth)

	runes := make([]rune, barWidth)
	for i := range runes {
		runes[i] = ' '
	}
	for i := 0; i < filled; i++ {
		runes[i] = '█'
	}
	if peak > 0 && peakPos < barWidth && peakPos >= filled {
		runes[peakPos] = '│'
	}

	fmt.Printf("\033[2K  %s [%s] %-12s  peak: %s\r\n",
		label,
		string(runes),
		formatBits(rate),
		formatBits(peak),
	)
}

func barFrac(value, maxValue int64, width int) int {
	if maxValue <= 0 || value <= 0 {
		return 0
	}
	n := int(float64(value) / float64(maxValue) * float64(width))
	if n > width {
		return width
	}
	return n
}

func formatBits(bps int64) string {
	switch {
	case bps >= 1_000_000_000:
		return fmt.Sprintf("%.2f Gbps", float64(bps)/1e9)
	case bps >= 1_000_000:
		return fmt.Sprintf("%.2f Mbps", float64(bps)/1e6)
	case bps >= 1_000:
		return fmt.Sprintf("%.2f Kbps", float64(bps)/1e3)
	default:
		return fmt.Sprintf("%d bps", bps)
	}
}
