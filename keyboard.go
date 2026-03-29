package main

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/term"
)

// handleKeyboard reads single keypresses from stdin (raw mode).
//   Q / q  — cancel context and exit
//   R / r  — reset peak counters
//   Ctrl+C — treated as quit (raw mode suppresses the OS signal for Ctrl+C,
//             so we handle byte 0x03 explicitly)
func handleKeyboard(cancel context.CancelFunc, stats *Stats) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return
	}
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			return
		}
		switch buf[0] {
		case 'q', 'Q', 3: // 3 = Ctrl+C
			fmt.Print("\r\nQuit.\r\n")
			cancel()
			return
		case 'r', 'R':
			stats.ResetPeaks()
		}
	}
}
