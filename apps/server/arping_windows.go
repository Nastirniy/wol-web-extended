// +build windows

package main

import (
	"context"
	"fmt"
	"net"
)

// ARPPingMAC is not supported on Windows - always returns error
// On Windows, the system falls back to passive ARP table lookups only
func ARPPingMAC(mac string, networkInterface string, timeoutSeconds int) (string, bool, error) {
	return "", false, fmt.Errorf("active ARP scanning is not supported on Windows")
}

// Stub helper functions for Windows (not used, but needed for compilation)

func scanNetworkForMAC(ctx context.Context, ipNet *net.IPNet, ifaceName string, targetMAC string) (string, bool) {
	return "", false
}

func getBroadcastIP(ipNet *net.IPNet) net.IP {
	ip := ipNet.IP.To4()
	mask := ipNet.Mask

	// Ensure we're working with IPv4
	if ip == nil {
		return nil
	}

	// If mask is IPv6 format, convert to IPv4
	if len(mask) == 16 {
		mask = mask[12:16]
	}

	broadcast := make(net.IP, len(ip))
	for i := 0; i < len(ip); i++ {
		broadcast[i] = ip[i] | ^mask[i]
	}
	return broadcast
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func duplicateIP(ip net.IP) net.IP {
	dup := make(net.IP, len(ip))
	copy(dup, ip)
	return dup
}
