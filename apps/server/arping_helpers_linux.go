//go:build linux
// +build linux

package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/j-keck/arping"
)

// ARPPingIP sends an ARP ping to a specific IP address and returns the hardware address
//
// IMPORTANT WARNING: When multiple interfaces are specified, this function tries each
// interface in sequence and returns on the FIRST successful response. This can cause
// incorrect results in network topologies with overlapping IP ranges:
//
// Example problem scenario:
//
//	eth0: 192.168.1.0/24 with device at 192.168.1.100 (MAC: AA:BB:CC:DD:EE:FF)
//	eth1: 192.168.1.0/24 with device at 192.168.1.100 (MAC: 11:22:33:44:55:66)
//
// If eth0 is tried first, it will succeed and return the wrong device's MAC address.
// The calling code MUST verify the returned MAC address matches the expected device.
//
// Best practice: Use non-overlapping IP ranges or per-host interface configuration.
func ARPPingIP(ip string, ifaceName string) (net.HardwareAddr, error) {
	// Determine interface description for logging
	ifaceDesc := "all available interfaces"
	if ifaceName != "" {
		interfaces := strings.Split(ifaceName, ",")
		if len(interfaces) == 1 {
			ifaceDesc = fmt.Sprintf("interface %s", strings.TrimSpace(interfaces[0]))
		} else {
			ifaceDesc = fmt.Sprintf("interfaces [%s]", ifaceName)
		}
	}

	Debug("ARP ping to IP %s using %s", ip, ifaceDesc)

	// If multiple interfaces specified (comma-separated), try each one
	// WARNING: Returns on first successful response - see function documentation
	if ifaceName != "" {
		interfaces := strings.Split(ifaceName, ",")
		for _, iface := range interfaces {
			iface = strings.TrimSpace(iface)
			if iface == "" {
				continue
			}

			Debug("Sending ARP request to %s via %s", ip, iface)
			hwAddr, duration, err := arping.PingOverIfaceByName(net.ParseIP(ip), iface)
			if err == nil && hwAddr != nil {
				Debug("ARP reply from %s: MAC %s via %s (%.3fms)",
					ip, hwAddr.String(), iface, duration.Seconds()*1000)
				// WARNING: Returning first successful response - caller must verify MAC address
				return hwAddr, nil
			}
			// Check for permission error
			if strings.Contains(err.Error(), "operation not permitted") {
				Warning("ARP ping requires CAP_NET_RAW capability. Run with: sudo setcap cap_net_raw+ep /path/to/wolweb")
				return nil, fmt.Errorf("ARP ping requires elevated permissions (CAP_NET_RAW)")
			}
			Debug("No ARP reply from %s via %s: %v", ip, iface, err)
		}
		Debug("ARP ping to %s failed on all specified interfaces, trying default", ip)
	}

	// If no interface specified or all failed, try default (no interface specified)
	Debug("Sending ARP request to %s via default interface", ip)
	hwAddr, duration, err := arping.Ping(net.ParseIP(ip))
	if err != nil {
		// Check for permission error
		if strings.Contains(err.Error(), "operation not permitted") {
			Warning("ARP ping requires CAP_NET_RAW capability. Run with: sudo setcap cap_net_raw+ep /path/to/wolweb")
			return nil, fmt.Errorf("ARP ping requires elevated permissions (CAP_NET_RAW)")
		}
		Debug("No ARP reply from %s via default interface: %v", ip, err)
		return nil, fmt.Errorf("ARP ping failed: %w", err)
	}

	Debug("ARP reply from %s: MAC %s via default interface (%.3fms)",
		ip, hwAddr.String(), duration.Seconds()*1000)
	return hwAddr, nil
}
