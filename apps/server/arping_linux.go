//go:build linux
// +build linux

package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/j-keck/arping"
)

// ARPPingMAC performs active ARP scanning to find and ping a host by MAC address
// Uses arping library to actively scan the network (Linux only)
func ARPPingMAC(mac string, networkInterface string, timeoutSeconds int) (string, bool, error) {
	// Normalize target MAC for comparison
	normalizedTargetMAC := normalizeMACAddress(mac)

	// Determine which interface(s) to use
	interfaces := []string{}
	if networkInterface != "" {
		interfaces = strings.Split(networkInterface, ",")
	} else {
		// Get all interfaces
		ifaces, err := net.Interfaces()
		if err != nil {
			return "", false, fmt.Errorf("failed to get interfaces: %w", err)
		}
		for _, iface := range ifaces {
			// Skip loopback and down interfaces
			if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
				interfaces = append(interfaces, iface.Name)
			}
		}
	}

	// Set arping timeout per IP (use a fraction of total timeout for individual pings)
	// This allows scanning multiple IPs within the timeout window
	arpTimeout := time.Duration(timeoutSeconds) * time.Second / 20 // 1/20th per IP for fast scanning
	if arpTimeout < 50*time.Millisecond {
		arpTimeout = 50 * time.Millisecond
	}
	arping.SetTimeout(arpTimeout)

	// Try each interface
	for _, ifaceName := range interfaces {
		ifaceName = strings.TrimSpace(ifaceName)
		if ifaceName == "" {
			continue
		}

		// Get interface to determine network
		iface, err := net.InterfaceByName(ifaceName)
		if err != nil {
			Debug("Failed to get interface %s: %v", ifaceName, err)
			continue
		}

		// Get interface addresses to determine network CIDR
		addrs, err := iface.Addrs()
		if err != nil {
			Debug("Failed to get addresses for interface %s: %v", ifaceName, err)
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil {
				continue
			}

			// Calculate network size
			ones, bits := ipNet.Mask.Size()
			networkSize := 1 << (bits - ones)
			Debug("Scanning network %s (%d hosts) on interface %s for MAC %s (timeout: %ds)",
				ipNet.String(), networkSize-2, ifaceName, mac, timeoutSeconds)

			// Create timeout context using provided timeout
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
			defer cancel()

			// Scan the network range using arping with timeout
			startTime := time.Now()
			foundIP, found := scanNetworkForMAC(ctx, ipNet, ifaceName, normalizedTargetMAC)
			scanDuration := time.Since(startTime)

			if found {
				Debug("MAC %s found at IP %s on interface %s after %.2fs",
					mac, foundIP, ifaceName, scanDuration.Seconds())

				// Verify with ICMP ping
				Debug("Verifying connectivity to %s with ICMP ping", foundIP)
				pingSuccess := PingHostWithInterface(foundIP, ARPPingTimeoutSeconds, ifaceName)
				if pingSuccess {
					Debug("ICMP ping to %s successful", foundIP)
				} else {
					Debug("ICMP ping to %s failed (host found via ARP but not responding to ICMP)", foundIP)
				}
				return foundIP, pingSuccess, nil
			}

			// Check if timeout occurred
			if ctx.Err() == context.DeadlineExceeded {
				Debug("ARP scan for MAC %s on interface %s timed out after %.2fs (network: %s)",
					mac, ifaceName, scanDuration.Seconds(), ipNet.String())
			} else {
				Debug("ARP scan for MAC %s on interface %s completed in %.2fs - not found (network: %s)",
					mac, ifaceName, scanDuration.Seconds(), ipNet.String())
			}
		}
	}

	return "", false, fmt.Errorf("host not found on any interface")
}

// scanNetworkForMAC scans a network range using ARP ping to find a specific MAC address
// Uses parallel scanning with timeout context for efficiency
func scanNetworkForMAC(ctx context.Context, ipNet *net.IPNet, ifaceName string, targetMAC string) (string, bool) {
	// Calculate network range - ensure IPv4
	ip := ipNet.IP.To4()
	if ip == nil {
		// Not IPv4, skip
		return "", false
	}

	ip = ip.Mask(ipNet.Mask)
	broadcast := getBroadcastIP(ipNet)
	if broadcast == nil {
		// Failed to calculate broadcast
		return "", false
	}

	// Create channels for results and IP distribution
	type arpResult struct {
		ip  string
		mac string
	}

	resultChan := make(chan arpResult, 10)
	ipChan := make(chan net.IP, 50)

	// Use worker pool for parallel scanning
	const numWorkers = 10
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case currentIP, ok := <-ipChan:
					if !ok {
						return
					}

					// Use arping to ping this IP
					hwAddr, _, err := arping.PingOverIfaceByName(currentIP, ifaceName)
					if err == nil && hwAddr != nil {
						select {
						case resultChan <- arpResult{ip: currentIP.String(), mac: hwAddr.String()}:
						case <-ctx.Done():
							return
						}
					} else if err != nil && strings.Contains(err.Error(), "operation not permitted") {
						// Permission error - stop scanning and return early
						Warning("ARP scanning requires CAP_NET_RAW capability. Run with: sudo setcap cap_net_raw+ep /path/to/wolweb")
						return
					}
				}
			}
		}()
	}

	// Start IP distributor
	go func() {
		defer close(ipChan)
		for currentIP := duplicateIP(ip); ipNet.Contains(currentIP); incIP(currentIP) {
			if currentIP.Equal(broadcast) || currentIP.Equal(ip) {
				continue
			}

			select {
			case <-ctx.Done():
				return
			case ipChan <- duplicateIP(currentIP):
			}
		}
	}()

	// Wait for workers in background
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Process results
	for {
		select {
		case <-ctx.Done():
			return "", false
		case result, ok := <-resultChan:
			if !ok {
				// All workers done, no match found
				return "", false
			}

			// Check if MAC matches
			resultMAC := normalizeMACAddress(result.mac)
			if resultMAC == targetMAC {
				return result.ip, true
			}
		}
	}
}

// getBroadcastIP calculates the broadcast address for a network
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

// incIP increments an IP address
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// duplicateIP creates a copy of an IP address
func duplicateIP(ip net.IP) net.IP {
	dup := make(net.IP, len(ip))
	copy(dup, ip)
	return dup
}
