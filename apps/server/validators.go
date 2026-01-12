package main

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// validateNetworkInterface validates that the provided interface name(s) is safe and exists
// Supports comma-separated list of interface names
func validateNetworkInterface(interfaceName string) error {
	// Empty interface means "all interfaces" which is always valid
	if interfaceName == "" {
		return nil
	}

	// Split by comma to handle multiple interfaces
	interfaces := strings.Split(interfaceName, ",")

	// Get available interfaces once
	availableInterfaces, err := net.Interfaces()
	if err != nil {
		return &ValidationError{Code: ErrCodeNetworkError, Message: fmt.Sprintf("failed to get available interfaces: %v", err)}
	}

	// Build a map of available interface names for faster lookup
	availableMap := make(map[string]bool)
	for _, iface := range availableInterfaces {
		availableMap[iface.Name] = true
	}

	// Validate each interface
	for _, iface := range interfaces {
		iface = strings.TrimSpace(iface)
		if iface == "" {
			continue // Skip empty entries from trailing commas
		}

		// Validate interface name format (allow alphanumeric, spaces, dots, dashes, underscores, parentheses)
		for _, char := range iface {
			if !((char >= 'a' && char <= 'z') ||
				(char >= 'A' && char <= 'Z') ||
				(char >= '0' && char <= '9') ||
				char == '.' || char == '-' || char == '_' ||
				char == ' ' || char == '(' || char == ')') {
				return &ValidationError{Code: ErrCodeInvalidInterface, Message: fmt.Sprintf("invalid interface name format: %s", iface)}
			}
		}

		// Check if interface actually exists
		if !availableMap[iface] {
			return &ValidationError{Code: ErrCodeInterfaceNotFound, Message: fmt.Sprintf("interface %s does not exist", iface)}
		}
	}

	return nil
}

// sanitizeIPAddress validates and sanitizes IP addresses
func sanitizeIPAddress(ip string) error {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return fmt.Errorf("IP address cannot be empty")
	}

	// Parse and validate IP address
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address format: %s", ip)
	}

	// Prevent access to localhost/loopback addresses for security
	if parsedIP.IsLoopback() {
		return fmt.Errorf("loopback addresses are not allowed")
	}

	// Prevent access to multicast addresses
	if parsedIP.IsMulticast() {
		return fmt.Errorf("multicast addresses are not allowed")
	}

	return nil
}

// sanitizeStaticIPv4 validates static IPv4 addresses for hosts
// This is used for the static_ip field which allows manual IP specification
func sanitizeStaticIPv4(ip string) error {
	ip = strings.TrimSpace(ip)

	// Empty is valid - means no static IP configured
	if ip == "" {
		return nil
	}

	// Parse and validate IP address
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return &ValidationError{Code: ErrCodeInvalidIP, Message: fmt.Sprintf("invalid IPv4 address format: %s", ip)}
	}

	// Ensure it's IPv4 (not IPv6)
	if parsedIP.To4() == nil {
		return &ValidationError{Code: ErrCodeInvalidIP, Message: fmt.Sprintf("only IPv4 addresses are supported: %s", ip)}
	}

	// Prevent access to localhost/loopback addresses
	if parsedIP.IsLoopback() {
		return &ValidationError{Code: ErrCodeInvalidIP, Message: "loopback addresses are not allowed"}
	}

	// Prevent multicast addresses
	if parsedIP.IsMulticast() {
		return &ValidationError{Code: ErrCodeInvalidIP, Message: "multicast addresses are not allowed"}
	}

	// Prevent broadcast address
	if ip == "255.255.255.255" {
		return &ValidationError{Code: ErrCodeInvalidIP, Message: "broadcast address is not allowed"}
	}

	// Prevent all-zero address
	if ip == "0.0.0.0" {
		return &ValidationError{Code: ErrCodeInvalidIP, Message: "all-zero address is not allowed"}
	}

	return nil
}

// sanitizeMACAddress validates and sanitizes MAC addresses
func sanitizeMACAddress(mac string) error {
	mac = strings.TrimSpace(mac)
	if mac == "" {
		return &ValidationError{Code: ErrCodeMissingField, Message: "MAC address cannot be empty"}
	}

	// Remove common separators for validation
	cleanMAC := strings.ReplaceAll(mac, ":", "")
	cleanMAC = strings.ReplaceAll(cleanMAC, "-", "")
	cleanMAC = strings.ReplaceAll(cleanMAC, " ", "")

	// Validate MAC address format (12 hex characters)
	macRegex := regexp.MustCompile(`^[0-9a-fA-F]{12}$`)
	if !macRegex.MatchString(cleanMAC) {
		return &ValidationError{Code: ErrCodeInvalidMAC, Message: fmt.Sprintf("invalid MAC address format: %s", mac)}
	}

	// Prevent broadcast MAC address
	if strings.ToLower(cleanMAC) == "ffffffffffff" {
		return &ValidationError{Code: ErrCodeInvalidMAC, Message: "broadcast MAC address is not allowed"}
	}

	// Prevent all-zero MAC address
	if cleanMAC == "000000000000" {
		return &ValidationError{Code: ErrCodeInvalidMAC, Message: "all-zero MAC address is not allowed"}
	}

	return nil
}

// sanitizeBroadcastAddress validates broadcast address format
func sanitizeBroadcastAddress(broadcast string) error {
	broadcast = strings.TrimSpace(broadcast)
	if broadcast == "" {
		return &ValidationError{Code: ErrCodeMissingField, Message: "broadcast address cannot be empty"}
	}

	// Validate format: IP:PORT
	parts := strings.Split(broadcast, ":")
	if len(parts) != 2 {
		return &ValidationError{Code: ErrCodeInvalidBroadcast, Message: "broadcast address must be in format IP:PORT"}
	}

	// Validate IP part
	if err := sanitizeIPAddress(parts[0]); err != nil {
		return &ValidationError{Code: ErrCodeInvalidBroadcast, Message: fmt.Sprintf("invalid broadcast IP: %v", err)}
	}

	// Validate port part (simple check)
	portRegex := regexp.MustCompile(`^[0-9]{1,5}$`)
	if !portRegex.MatchString(parts[1]) {
		return &ValidationError{Code: ErrCodeInvalidBroadcast, Message: "invalid port format in broadcast address"}
	}

	return nil
}

// sanitizeHostName validates host names
func sanitizeHostName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return &ValidationError{Code: ErrCodeMissingField, Message: "host name cannot be empty"}
	}

	// Limit length
	if len(name) > 64 {
		return &ValidationError{Code: ErrCodeNameTooLong, Message: "host name too long (max 64 characters)"}
	}

	// Allow unicode letters/numbers, hyphens, dots, underscores, and spaces
	nameRegex := regexp.MustCompile(`^[\p{L}\p{N}\-\._\s]+$`)
	if !nameRegex.MatchString(name) {
		return &ValidationError{Code: ErrCodeInvalidInput, Message: "host name contains invalid characters"}
	}

	return nil
}

// normalizeMACAddress converts MAC address to lowercase with colon separators
func normalizeMACAddress(mac string) string {
	// Remove all separators (colons, hyphens, spaces)
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, " ", "")

	// Convert to lowercase
	mac = strings.ToLower(mac)

	// Validate length
	if len(mac) != 12 {
		return mac // Return as-is if invalid length
	}

	// Add colons every 2 characters
	result := make([]string, 6)
	for i := 0; i < 6; i++ {
		result[i] = mac[i*2 : i*2+2]
	}

	return strings.Join(result, ":")
}
