package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	userKey := "anonymous"
	if user != nil {
		userKey = user.ID
	}

	if !s.PingRateLimit.Allow(userKey) {
		response := map[string]interface{}{
			"ping_success": false,
			"arp_success":  false,
			"error":        "Rate limit exceeded. Please wait before making more ping requests.",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(response)
		return
	}

	var data struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		response := map[string]interface{}{
			"ping_success": false,
			"arp_success":  false,
			"error":        "Failed to read request data",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if data.ID == "" {
		response := map[string]interface{}{
			"ping_success": false,
			"arp_success":  false,
			"error":        "Host ID is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var host Host
	var query string
	var args []interface{}

	if s.Config.UseAuth && user != nil {
		query = "SELECT id, name, mac, broadcast, interface, static_ip, use_as_fallback, user_id FROM hosts WHERE id = ? AND user_id = ?"
		args = []interface{}{data.ID, user.ID}
	} else {
		// In no-auth mode, ONLY allow access to hosts with NULL user_id
		query = "SELECT id, name, mac, broadcast, interface, static_ip, use_as_fallback, user_id FROM hosts WHERE id = ? AND user_id IS NULL"
		args = []interface{}{data.ID}
	}

	err := s.DB.QueryRow(query, args...).Scan(&host.ID, &host.Name, &host.MAC, &host.Broadcast, &host.Interface, &host.StaticIP, &host.UseAsFallback, &host.UserID)
	if err == sql.ErrNoRows {
		// In no-auth mode, check if host exists but has a user_id
		if !s.Config.UseAuth {
			var exists bool
			err := s.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM hosts WHERE id = ?)", data.ID).Scan(&exists)
			if err == nil && exists {
				response := map[string]interface{}{
					"ping_success": false,
					"arp_success":  false,
					"error":        "Access denied: host not accessible in no-auth mode",
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(response)
				return
			}
		}
		response := map[string]interface{}{
			"ping_success": false,
			"arp_success":  false,
			"error":        "Host not found",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}
	if err != nil {
		response := map[string]interface{}{
			"ping_success": false,
			"arp_success":  false,
			"error":        "Failed to find host",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check cache first
	cacheKey := host.ID
	if cachedEntry := s.PingCache.Get(cacheKey); cachedEntry != nil {
		Debug("Cache HIT for host '%s' (MAC: %s) - returning cached result", host.Name, host.MAC)
		response := map[string]interface{}{
			"ping_success": cachedEntry.PingSuccess,
			"arp_success":  cachedEntry.ARPSuccess,
			"cached":       true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if ping is already in progress (request coalescing)
	isFirstRequest, waitChan := s.PingCache.StartPing(cacheKey)
	if !isFirstRequest {
		// Another request is already pinging this host - wait for result
		Debug("Ping already in progress for host '%s' (MAC: %s) - coalescing request", host.Name, host.MAC)

		select {
		case result := <-waitChan:
			if result.Error != nil {
				response := map[string]interface{}{
					"ping_success": false,
					"arp_success":  false,
					"error":        result.Error.Error(),
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(response)
				return
			}
			response := map[string]interface{}{
				"ping_success": result.PingSuccess,
				"arp_success":  result.ARPSuccess,
				"coalesced":    true,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		case <-time.After(time.Duration(s.Config.PingTimeout+5) * time.Second):
			// Timeout waiting for result
			response := map[string]interface{}{
				"ping_success": false,
				"arp_success":  false,
				"error":        "Timeout waiting for ping result",
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestTimeout)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// Determine which network interface(s) to use
	interfaceToUse := s.determineNetworkInterface(host)

	// Log interface configuration
	ifaceDesc := "all available interfaces"
	if interfaceToUse != "" {
		ifaceDesc = interfaceToUse
	}
	Debug("Checking status of host '%s' (MAC: %s) using %s", host.Name, host.MAC, ifaceDesc)

	var pingSuccess bool
	var arpSuccess bool

	// Check if static IP is configured
	if host.StaticIP != "" && !host.UseAsFallback {
		// Use static IP directly (ignore ARP resolution)
		Debug("Using configured static IP %s for host '%s' (MAC: %s)", host.StaticIP, host.Name, host.MAC)

		// Verify host is online using ARP ping
		hwAddr, err := ARPPingIP(host.StaticIP, interfaceToUse)
		if err == nil && hwAddr != nil {
			// Host responded to ARP ping
			arpSuccess = true
			pingSuccess = true
			Debug("Host '%s' is ONLINE at static IP %s (MAC: %s verified)", host.Name, host.StaticIP, hwAddr.String())

			// Verify MAC matches (warning if mismatch)
			detectedMAC := normalizeMACAddress(hwAddr.String())
			storedMAC := normalizeMACAddress(host.MAC)
			if detectedMAC != storedMAC {
				Warning("Host '%s' MAC mismatch - stored: %s, detected: %s at static IP %s",
					host.Name, storedMAC, detectedMAC, host.StaticIP)
				Warning("This may indicate network complexity (overlapping IP ranges, VLAN issues, or incorrect static IP configuration)")
			}
		} else {
			// Static IP not responding
			Debug("Host '%s' (MAC: %s) not responding at static IP %s", host.Name, host.MAC, host.StaticIP)
			pingSuccess = false
			arpSuccess = false
		}

		// Store result in cache
		s.PingCache.Set(cacheKey, pingSuccess, arpSuccess)

		response := map[string]interface{}{
			"ping_success": pingSuccess,
			"arp_success":  arpSuccess,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Try to resolve IP from MAC first (passive ARP table lookup)
	Debug("Looking up IP for MAC %s in ARP table", host.MAC)
	hostIP, ipErr := GetIPFromMAC(host.MAC)

	// For manual ping, flush ARP cache if entry exists to ensure fresh data
	if ipErr == nil && hostIP != "" {
		FlushARPEntryIfExists(hostIP)
		Debug("Manual ping - flushed ARP cache for IP %s to ensure fresh data", hostIP)
		// Re-lookup after flush
		hostIP, ipErr = GetIPFromMAC(host.MAC)
	}

	if ipErr == nil {
		// IP found in ARP table - now verify host is actually online using ARP ping
		Debug("Host '%s' (MAC: %s) found in ARP table at IP %s", host.Name, host.MAC, hostIP)
		Debug("Verifying host '%s' is online with active ARP ping", host.Name)

		// Use arping to actively verify the host responds
		hwAddr, err := ARPPingIP(hostIP, interfaceToUse)
		if err == nil && hwAddr != nil {
			// Host responded to ARP ping
			arpSuccess = true
			pingSuccess = true
			Debug("Host '%s' is ONLINE at IP %s (MAC: %s verified)", host.Name, hostIP, hwAddr.String())

			// Verify MAC matches
			detectedMAC := normalizeMACAddress(hwAddr.String())
			storedMAC := normalizeMACAddress(host.MAC)
			if detectedMAC != storedMAC {
				Warning("Host '%s' MAC mismatch - stored: %s, detected: %s at IP %s",
					host.Name, storedMAC, detectedMAC, hostIP)
			}
		} else {
			// Host in ARP table but not responding - flush stale entry
			FlushARPEntryIfExists(hostIP)
			Debug("Host '%s' (MAC: %s) in ARP table but not responding - flushed cache entry for IP %s", host.Name, host.MAC, hostIP)
			pingSuccess = false
			arpSuccess = false
		}
	} else {
		// IP not in ARP table - do full network scan to find host by MAC
		Debug("Host '%s' (MAC: %s) not in ARP table", host.Name, host.MAC)
		Debug("Starting full network ARP scan for host '%s' (timeout: %ds)", host.Name, s.Config.PingTimeout)

		foundIP, pingOk, arpErr := ARPPingMAC(host.MAC, interfaceToUse, s.Config.PingTimeout)
		if arpErr == nil {
			// Active ARP scan succeeded - host found
			pingSuccess = pingOk
			arpSuccess = true
			Debug("Host '%s' (MAC: %s) found via network scan at IP %s - status: %s",
				host.Name, host.MAC, foundIP, func() string {
					if pingOk {
						return "ONLINE"
					}
					return "FOUND"
				}())
		} else {
			// Host not found - try static IP as fallback if configured
			if host.StaticIP != "" && host.UseAsFallback {
				Debug("ARP resolution failed for host '%s', trying static IP %s as fallback", host.Name, host.StaticIP)

				hwAddr, fallbackErr := ARPPingIP(host.StaticIP, interfaceToUse)
				if fallbackErr == nil && hwAddr != nil {
					// Host responded at static IP
					arpSuccess = true
					pingSuccess = true
					Debug("Host '%s' is ONLINE at fallback static IP %s (MAC: %s)", host.Name, host.StaticIP, hwAddr.String())

					// Verify MAC matches
					detectedMAC := normalizeMACAddress(hwAddr.String())
					storedMAC := normalizeMACAddress(host.MAC)
					if detectedMAC != storedMAC {
						Warning("Host '%s' MAC mismatch - stored: %s, detected: %s at fallback static IP %s",
							host.Name, storedMAC, detectedMAC, host.StaticIP)
					}
				} else {
					Debug("Host '%s' (MAC: %s) NOT FOUND on network or at fallback static IP - OFFLINE", host.Name, host.MAC)
					s.PingCache.Set(cacheKey, false, false)
					response := map[string]interface{}{
						"ping_success": false,
						"arp_success":  false,
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(response)
					return
				}
			} else {
				Debug("Host '%s' (MAC: %s) NOT FOUND on network - OFFLINE", host.Name, host.MAC)
				s.PingCache.Set(cacheKey, false, false)
				response := map[string]interface{}{
					"ping_success": false,
					"arp_success":  false,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}
		}
	}

	Debug("Host '%s' final status - ping_success: %v, arp_success: %v", host.Name, pingSuccess, arpSuccess)

	// Store result in cache
	s.PingCache.Set(cacheKey, pingSuccess, arpSuccess)

	response := map[string]interface{}{
		"ping_success": pingSuccess,
		"arp_success":  arpSuccess,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
