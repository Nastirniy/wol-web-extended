package main

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
)

// PingHost pings a host and returns true if it's reachable
// Uses Linux ping command format
func PingHost(host string, timeout int) bool {
	cmd := exec.Command("ping", "-c", "1", "-W", fmt.Sprintf("%d", timeout), host)
	err := cmd.Run()
	return err == nil
}

// PingHostWithInterface pings a host using a specific network interface
//
// WARNING: When multiple interfaces are specified with overlapping IP ranges,
// this function may succeed on the wrong network segment. See arping_helpers_linux.go
// ARPPingIP function documentation for details about this limitation.
func PingHostWithInterface(host string, timeout int, networkInterface string) bool {
	// If no interface specified, use default ping
	if networkInterface == "" {
		return PingHost(host, timeout)
	}

	// Parse multiple interfaces (comma-separated) - tries each until success
	// WARNING: Returns on first successful ping - may be wrong device if IPs overlap
	interfaces := strings.Split(networkInterface, ",")
	for _, iface := range interfaces {
		iface = strings.TrimSpace(iface)
		if iface == "" {
			continue
		}

		// Try ping with this specific interface
		cmd := exec.Command("ping", "-c", "1", "-W", fmt.Sprintf("%d", timeout), "-I", iface, host)
		if err := cmd.Run(); err == nil {
			return true // Success with this interface - WARNING: see function documentation
		}
	}

	// If all interface-specific pings failed, fallback to default ping
	return PingHost(host, timeout)
}

// GetMACFromARP tries to get MAC address from ARP table for a given IP
// Uses Linux arp command format
func GetMACFromARP(ip string) (string, error) {
	cmd := exec.Command("arp", "-n", ip)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run arp command: %w", err)
	}

	// Parse ARP output to extract MAC address
	return parseMACFromARPOutput(string(output))
}

// GetIPFromMAC tries to find IP address from MAC in ARP table (passive lookup only)
// Uses Linux arp command format
func GetIPFromMAC(mac string) (string, error) {
	// Normalize the MAC address for comparison
	targetMAC := normalizeMACAddress(mac)

	cmd := exec.Command("arp", "-n")

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run arp command: %w", err)
	}

	// Parse full ARP table to find IP for this MAC
	return parseIPFromARPOutput(string(output), targetMAC)
}

// parseIPFromARPOutput scans ARP table for an IP with matching MAC
// Uses Linux ARP format: "192.168.1.100 ether aa:bb:cc:dd:ee:ff C eth0"
func parseIPFromARPOutput(output, targetMAC string) (string, error) {
	lines := strings.Split(output, "\n")

	ipRegex := regexp.MustCompile(`(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`)
	macRegex := regexp.MustCompile(`([0-9a-fA-F]{2}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2})`)

	for _, line := range lines {
		// Skip empty lines and headers
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Find MAC in this line
		macMatches := macRegex.FindStringSubmatch(line)
		if len(macMatches) < 2 {
			continue
		}

		// Normalize found MAC
		foundMAC := normalizeMACAddress(macMatches[1])

		// Check if this is the MAC we're looking for
		if foundMAC == targetMAC {
			// Find IP in this line
			ipMatches := ipRegex.FindStringSubmatch(line)
			if len(ipMatches) >= 2 {
				return ipMatches[1], nil
			}
		}
	}

	return "", fmt.Errorf("IP address not found for MAC %s in ARP table", targetMAC)
}

// parseMACFromARPOutput parses MAC address from Linux ARP output
// Linux ARP format: "192.168.1.100 ether aa:bb:cc:dd:ee:ff C eth0"
func parseMACFromARPOutput(output string) (string, error) {
	lines := strings.Split(output, "\n")

	macRegex := regexp.MustCompile(`([0-9a-fA-F]{2}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2})`)

	for _, line := range lines {
		if matches := macRegex.FindStringSubmatch(line); len(matches) > 1 {
			return strings.ToLower(matches[1]), nil
		}
	}

	return "", fmt.Errorf("MAC address not found in ARP table")
}

// SendWakeOnLan sends a WOL packet using the specified network interface(s)
// When multiple interfaces are specified, broadcasts to ALL of them (not just first successful)
func SendWakeOnLanWithInterface(mac, targetIP string, port int, networkInterfaces string) error {
	Debug("SendWakeOnLan called: MAC=%s, Target=%s:%d, Interfaces=%s",
		mac, targetIP, port, func() string {
			if networkInterfaces == "" {
				return "(all)"
			}
			return networkInterfaces
		}())

	// If no specific interface is specified, use the default behavior
	if networkInterfaces == "" {
		Debug("Using default WoL behavior (all interfaces)")
		return sendWakeOnLanDefault(mac, targetIP, port)
	}

	// Parse multiple interfaces (comma-separated)
	interfaces := strings.Split(networkInterfaces, ",")
	var errors []string
	successCount := 0

	Debug("Attempting to broadcast WoL packet via %d interface(s)", len(interfaces))

	for _, networkInterface := range interfaces {
		networkInterface = strings.TrimSpace(networkInterface)
		if networkInterface == "" {
			continue
		}

		Debug("Trying interface: %s", networkInterface)

		// Get the interface
		iface, err := net.InterfaceByName(networkInterface)
		if err != nil {
			errMsg := fmt.Sprintf("interface %s not found: %v", networkInterface, err)
			errors = append(errors, errMsg)
			Error("%s", errMsg)
			continue
		}

		// Get interface addresses
		addrs, err := iface.Addrs()
		if err != nil {
			errMsg := fmt.Sprintf("failed to get addresses for interface %s: %v", networkInterface, err)
			errors = append(errors, errMsg)
			Error("%s", errMsg)
			continue
		}

		var localIP net.IP
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ipNet.IP.To4() != nil {
					localIP = ipNet.IP
					break
				}
			}
		}

		if localIP == nil {
			errMsg := fmt.Sprintf("no IPv4 address found on interface %s", networkInterface)
			errors = append(errors, errMsg)
			Error("%s", errMsg)
			continue
		}

		Debug("Found local IP %s on interface %s", localIP.String(), networkInterface)

		// Send from this interface (continue to other interfaces even on success)
		err = sendWakeOnLanFromIP(mac, targetIP, port, localIP.String())
		if err == nil {
			successCount++
			Debug("Success WoL packet sent via interface %s (IP: %s) to %s:%d for MAC %s",
				networkInterface, localIP.String(), targetIP, port, mac)
		} else {
			errMsg := fmt.Sprintf("failed to send via interface %s: %v", networkInterface, err)
			errors = append(errors, errMsg)
			Error("%s", errMsg)
		}
	}

	// Return success if at least one interface succeeded
	if successCount > 0 {
		Debug("Success WoL packet broadcast to %d/%d interface(s) for MAC %s",
			successCount, len(interfaces), mac)
		return nil
	}

	// All interfaces failed
	if len(errors) > 0 {
		Error("WoL packet failed on all %d interface(s). Errors: %v", len(interfaces), strings.Join(errors, "; "))
		return fmt.Errorf("all interfaces failed: %s", strings.Join(errors, "; "))
	}
	return fmt.Errorf("no valid interfaces found")
}

// sendWakeOnLanDefault uses the default WOL implementation
func sendWakeOnLanDefault(mac, targetIP string, port int) error {
	Debug("sendWakeOnLanDefault: Trying all available interfaces")

	// Create magic packet
	magicPacket, err := createMagicPacket(mac)
	if err != nil {
		Error("Failed to create magic packet for MAC %s: %v", mac, err)
		return fmt.Errorf("failed to create magic packet: %w", err)
	}

	// Resolve the broadcast address using udp4 to ensure IPv4
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", targetIP, port))
	if err != nil {
		Error("Failed to resolve UDP address %s:%d: %v", targetIP, port, err)
		return fmt.Errorf("failed to resolve UDP address %s:%d: %w", targetIP, port, err)
	}

	// Get a list of all network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		Error("Failed to get network interfaces: %v", err)
		return fmt.Errorf("failed to get network interfaces: %w", err)
	}

	Debug("Found %d network interfaces", len(interfaces))

	// Try to send from each active interface
	var lastErr error
	var triedInterfaces []string
	successCount := 0

	for _, iface := range interfaces {
		// Skip down or loopback interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			Debug("Skipping interface %s (down or loopback)", iface.Name)
			continue
		}

		// Get addresses for this interface
		addrs, err := iface.Addrs()
		if err != nil {
			Debug("Failed to get addresses for interface %s: %v", iface.Name, err)
			continue
		}

		// Try each IPv4 address on this interface
		for _, ifaceAddr := range addrs {
			ipNet, ok := ifaceAddr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil {
				continue
			}

			triedInterfaces = append(triedInterfaces, fmt.Sprintf("%s(%s)", iface.Name, ipNet.IP.String()))

			// Bind to this local IP
			localAddr := &net.UDPAddr{
				IP:   ipNet.IP,
				Port: 0,
			}

			Debug("Attempting to send from interface %s (IP: %s)", iface.Name, ipNet.IP.String())

			// Create UDP connection bound to this specific interface
			conn, err := net.DialUDP("udp4", localAddr, addr)
			if err != nil {
				lastErr = err
				Debug("Failed to create UDP connection from %s: %v", ipNet.IP.String(), err)
				continue
			}

			// Try to send the packet
			_, err = conn.Write(magicPacket)
			conn.Close()

			if err == nil {
				// Success! Packet sent
				successCount++
				Debug("Successfully sent WoL packet from interface %s (IP: %s)", iface.Name, ipNet.IP.String())
			} else {
				lastErr = err
				Debug("Failed to send from interface %s (IP: %s): %v", iface.Name, ipNet.IP.String(), err)
			}
		}
	}

	// If at least one send succeeded, return success
	if successCount > 0 {
		Debug("Success WoL packet sent from %d interface(s) to %s:%d for MAC %s",
			successCount, targetIP, port, mac)
		return nil
	}

	// If we got here, all interfaces failed
	if lastErr != nil {
		Error("Failed to send WoL packet from any interface. Tried: %v. Last error: %v",
			triedInterfaces, lastErr)
		return fmt.Errorf("failed to send magic packet from any interface (tried %d): %w", len(triedInterfaces), lastErr)
	}

	Error("No suitable network interface found for sending WoL packet")
	return fmt.Errorf("no suitable network interface found")
}

// sendWakeOnLanFromIP sends WOL packet from a specific local IP
func sendWakeOnLanFromIP(mac, targetIP string, port int, localIP string) error {
	magicPacket, err := createMagicPacket(mac)
	if err != nil {
		return err
	}

	// Get all network interfaces to find the correct one
	interfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to get network interfaces: %w", err)
	}

	// Find the interface with the matching local IP
	var targetInterface *net.Interface
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				if ipNet.IP.String() == localIP {
					targetInterface = &iface
					break
				}
			}
		}
		if targetInterface != nil {
			break
		}
	}

	if targetInterface == nil {
		return fmt.Errorf("could not find interface for IP %s", localIP)
	}

	// Create UDP address for local binding
	localAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:0", localIP))
	if err != nil {
		return fmt.Errorf("failed to resolve local address: %w", err)
	}

	// Create UDP address for target
	targetAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", targetIP, port))
	if err != nil {
		return fmt.Errorf("failed to resolve target address: %w", err)
	}

	// Create UDP connection bound to specific interface
	conn, err := net.DialUDP("udp4", localAddr, targetAddr)
	if err != nil {
		return fmt.Errorf("failed to create UDP connection: %w", err)
	}
	defer conn.Close()

	_, err = conn.Write(magicPacket)
	if err != nil {
		return fmt.Errorf("failed to send magic packet: %w", err)
	}

	return nil
}

// createMagicPacket creates a Wake-on-LAN magic packet
func createMagicPacket(mac string) ([]byte, error) {
	// Remove colons and hyphens from MAC address
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")

	if len(mac) != 12 {
		return nil, fmt.Errorf("invalid MAC address format: %s", mac)
	}

	// Convert MAC address to bytes
	macBytes := make([]byte, 6)
	for i := 0; i < 6; i++ {
		byteVal := 0
		for j := 0; j < 2; j++ {
			char := mac[i*2+j]
			var digit int
			if char >= '0' && char <= '9' {
				digit = int(char - '0')
			} else if char >= 'a' && char <= 'f' {
				digit = int(char - 'a' + 10)
			} else if char >= 'A' && char <= 'F' {
				digit = int(char - 'A' + 10)
			} else {
				return nil, fmt.Errorf("invalid character in MAC address: %c", char)
			}
			byteVal = byteVal*16 + digit
		}
		macBytes[i] = byte(byteVal)
	}

	// Create magic packet: 6 bytes of 0xFF followed by 16 repetitions of the MAC address
	packet := make([]byte, 102) // 6 + 16*6 = 102 bytes

	// Fill first 6 bytes with 0xFF
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}

	// Repeat MAC address 16 times
	for i := 0; i < 16; i++ {
		copy(packet[6+i*6:6+(i+1)*6], macBytes)
	}

	return packet, nil
}
