package main

import (
	"fmt"
	"strconv"
	"strings"
)

// parsePorts parses a port range string such as "1024-1042" or "1024,1025,1030-1042"
// into a slice of individual port numbers.
func parsePorts(s string) ([]uint16, error) {
	var ports []uint16
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if dash := strings.Index(part, "-"); dash >= 0 {
			loStr := strings.TrimSpace(part[:dash])
			hiStr := strings.TrimSpace(part[dash+1:])
			lo, err := strconv.ParseUint(loStr, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid port %q", loStr)
			}
			hi, err := strconv.ParseUint(hiStr, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid port %q", hiStr)
			}
			if lo > hi {
				return nil, fmt.Errorf("invalid range: %d > %d", lo, hi)
			}
			for p := lo; p <= hi; p++ {
				ports = append(ports, uint16(p))
			}
		} else {
			p, err := strconv.ParseUint(part, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid port %q", part)
			}
			ports = append(ports, uint16(p))
		}
	}
	return ports, nil
}

func portInList(ports []uint16, port uint16) bool {
	for _, p := range ports {
		if p == port {
			return true
		}
	}
	return false
}
