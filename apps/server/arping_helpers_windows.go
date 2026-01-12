// +build windows

package main

import (
	"fmt"
	"net"
)

// ARPPingIP is not supported on Windows
func ARPPingIP(ip string, ifaceName string) (net.HardwareAddr, error) {
	return nil, fmt.Errorf("ARP ping is not supported on Windows")
}
