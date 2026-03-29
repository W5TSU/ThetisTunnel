package main

import (
	"fmt"
	"sync"
)

const logMaxLines = 6

// UILogger collects status messages for the UI log area.
// When noUI is true, or when the receiver is nil, messages are printed
// directly to stdout instead.
// When debug is true, Debugf calls are also emitted.
type UILogger struct {
	mu    sync.Mutex
	lines []string
	noUI  bool
	debug bool
}

func newUILogger(noUI, debug bool) *UILogger {
	return &UILogger{noUI: noUI, debug: debug}
}

func (l *UILogger) Log(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if l == nil || l.noUI {
		fmt.Println(msg)
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, msg)
	if len(l.lines) > logMaxLines {
		l.lines = l.lines[len(l.lines)-logMaxLines:]
	}
}

// Debugf logs a message only when debug mode is enabled.
func (l *UILogger) Debugf(format string, args ...any) {
	if l == nil || !l.debug {
		return
	}
	l.Log("[DBG] "+format, args...)
}

// Lines returns a snapshot of the most recent log entries.
func (l *UILogger) Lines() []string {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.lines))
	copy(out, l.lines)
	return out
}
