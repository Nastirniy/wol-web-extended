//go:build linux
// +build linux

package main

import (
	"fmt"
	"os/exec"
)

// FlushARPEntry removes a specific IP from the ARP cache
// Uses 'ip neigh flush' command (modern replacement for 'arp -d')
// Requires CAP_NET_ADMIN capability in Docker
func FlushARPEntry(ip string) error {
	if ip == "" {
		return fmt.Errorf("IP address is required")
	}

	Debug("Flushing ARP cache entry for IP %s", ip)

	// Use 'ip neigh flush' instead of 'arp -d' (more reliable, modern approach)
	cmd := exec.Command("ip", "neigh", "flush", "to", ip)
	output, err := cmd.CombinedOutput()

	if err != nil {
		Warning("Failed to flush ARP entry for %s: %v (output: %s)", ip, err, string(output))
		return fmt.Errorf("failed to flush ARP entry: %w", err)
	}

	Debug("Successfully flushed ARP cache entry for IP %s", ip)
	return nil
}

// FlushARPEntryIfExists flushes ARP entry without failing if IP not found
// This is safe to call even if the IP doesn't exist in the ARP cache
func FlushARPEntryIfExists(ip string) {
	if err := FlushARPEntry(ip); err != nil {
		// Log but don't fail - entry might not exist yet or we lack permissions
		Debug("ARP flush for %s: %v", ip, err)
	}
}
