//go:build windows

package main

import (
	"context"
	"net"
	"syscall"
)

// listenPacketBroadcast opens a UDP PacketConn with SO_BROADCAST enabled
// so the proxy socket can send discovery frames to broadcast addresses.
func listenPacketBroadcast(ctx context.Context, addr string) (net.PacketConn, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var innerErr error
			if err := c.Control(func(fd uintptr) {
				innerErr = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
			}); err != nil {
				return err
			}
			return innerErr
		},
	}
	return lc.ListenPacket(ctx, "udp4", addr)
}
