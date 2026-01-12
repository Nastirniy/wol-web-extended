//go:build windows
// +build windows

package main

// FlushARPEntry is a no-op on Windows
// Windows ARP cache management requires different commands and is not implemented
func FlushARPEntry(ip string) error {
	Debug("ARP flush not supported on Windows (IP: %s)", ip)
	return nil
}

// FlushARPEntryIfExists is a no-op on Windows
func FlushARPEntryIfExists(ip string) {
	// No-op on Windows
}
