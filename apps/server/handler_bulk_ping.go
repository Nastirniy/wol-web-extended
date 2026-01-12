package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

// handleBulkPing handles POST requests for bulk pinging all hosts
//
// Bulk ping uses streaming SSE-style response to send results immediately as they arrive.
// This provides better user experience than waiting for all pings to complete.
//
// Features:
// - Concurrent ping execution for all hosts
// - Streaming results as they complete (no waiting for slowest host)
// - Request coalescing (multiple requests for same host share ping operation)
// - Cache integration (use cached results if available)
// - Prioritizes recently woken hosts (via WoL) for better UX
//
// Rate limiting: Dynamic based on host count, minimum 10 requests per timeout window
//
// This approach provides better UX than batch ping as users see results immediately
// rather than waiting for all pings to complete.
func (s *Server) handleBulkPing(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	userKey := "anonymous"
	if user != nil {
		userKey = user.ID
	}

	// Get user's hosts for rate limiting calculation
	var hostCount int
	var query string
	var args []interface{}

	if s.Config.UseAuth && user != nil {
		query = "SELECT COUNT(*) FROM hosts WHERE user_id = ? "
		args = []interface{}{user.ID}
	} else {
		query = "SELECT COUNT(*) FROM hosts WHERE user_id IS NULL"
		args = []interface{}{}
	}

	err := s.DB.QueryRow(query, args...).Scan(&hostCount)
	if err != nil {
		hostCount = 0
	}

	// Dynamic rate limiting: hosts_count * multiplier pings per timeout window
	dynamicLimit := hostCount * BulkPingMultiplier
	if dynamicLimit < MinBulkPingLimit {
		dynamicLimit = MinBulkPingLimit
	}

	// Use hardcoded time window from ping timeout config
	pingWindow := time.Duration(s.Config.PingTimeout) * time.Second

	// Check rate limit with dynamic limit
	if !s.checkDynamicRateLimit(userKey, dynamicLimit, pingWindow) {
		response := map[string]interface{}{
			"error": "Rate limit exceeded. Please wait before making more ping requests.",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get all hosts in WoL priority order
	var hosts []Host
	var rows *sql.Rows

	if s.Config.UseAuth && user != nil {
		rows, err = s.DB.Query("SELECT id, name, mac, broadcast, interface, static_ip, use_as_fallback, user_id, created, updated FROM hosts WHERE user_id = ?", user.ID)
	} else {
		rows, err = s.DB.Query("SELECT id, name, mac, broadcast, interface, static_ip, use_as_fallback, user_id, created, updated FROM hosts WHERE user_id IS NULL")
	}

	if err != nil {
		sendJSONError(w, "Failed to fetch hosts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var host Host
		err := rows.Scan(&host.ID, &host.Name, &host.MAC, &host.Broadcast, &host.Interface, &host.StaticIP, &host.UseAsFallback, &host.UserID, &host.Created, &host.Updated)
		if err != nil {
			continue
		}
		hosts = append(hosts, host)
	}

	// Sort hosts by WoL priority (recent WoL first)
	hosts = s.WoLHistory.SortHostsByWoLPriority(hosts)

	// Stream ping results as they come in
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	encoder := json.NewEncoder(w)
	flusher, ok := w.(http.Flusher)
	if !ok {
		sendJSONError(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Start response array
	w.Write([]byte("["))

	// Channel to collect ping results
	type pingResult struct {
		index  int
		result map[string]interface{}
	}
	resultChan := make(chan pingResult, len(hosts))

	// Execute pings in parallel using goroutines
	for i, host := range hosts {
		go func(idx int, h Host) {
			cacheKey := h.ID
			var pingSuccess bool
			var arpSuccess bool

			// Check cache first
			if cachedEntry := s.PingCache.Get(cacheKey); cachedEntry != nil {
				Debug("Bulk ping - Cache HIT for host '%s'", h.Name)
				result := map[string]interface{}{
					"host_id":      h.ID,
					"host_name":    h.Name,
					"ping_success": cachedEntry.PingSuccess,
					"arp_success":  cachedEntry.ARPSuccess,
				}
				resultChan <- pingResult{index: idx, result: result}
				return
			}

			// Check if ping is already in progress (request coalescing)
			isFirstRequest, waitChan := s.PingCache.StartPing(cacheKey)
			if !isFirstRequest {
				// Another request is already pinging this host - wait for result
				Debug("Bulk ping - Ping in progress for host '%s', coalescing", h.Name)
				select {
				case res := <-waitChan:
					result := map[string]interface{}{
						"host_id":      h.ID,
						"host_name":    h.Name,
						"ping_success": res.PingSuccess,
						"arp_success":  res.ARPSuccess,
					}
					resultChan <- pingResult{index: idx, result: result}
					return
				case <-time.After(time.Duration(s.Config.PingTimeout+5) * time.Second):
					// Timeout - return offline
					result := map[string]interface{}{
						"host_id":      h.ID,
						"host_name":    h.Name,
						"ping_success": false,
						"arp_success":  false,
					}
					resultChan <- pingResult{index: idx, result: result}
					return
				}
			}

			// Determine which network interface(s) to use
			interfaceToUse := s.determineNetworkInterface(h)

			// Check if static IP is configured (not fallback)
			if h.StaticIP != "" && !h.UseAsFallback {
				// Use static IP directly
				hwAddr, err := ARPPingIP(h.StaticIP, interfaceToUse)
				if err == nil && hwAddr != nil {
					arpSuccess = true
					pingSuccess = true
				} else {
					pingSuccess = false
					arpSuccess = false
				}

				s.PingCache.Set(cacheKey, pingSuccess, arpSuccess)
				result := map[string]interface{}{
					"host_id":      h.ID,
					"host_name":    h.Name,
					"ping_success": pingSuccess,
					"arp_success":  arpSuccess,
				}
				resultChan <- pingResult{index: idx, result: result}
				return
			}

			// Try to resolve IP from MAC first (passive ARP table lookup)
			hostIP, ipErr := GetIPFromMAC(h.MAC)

			if ipErr == nil {
				// IP found in ARP table - now verify host is actually online using ARP ping
				hwAddr, err := ARPPingIP(hostIP, interfaceToUse)
				if err == nil && hwAddr != nil {
					// Host responded to ARP ping
					arpSuccess = true
					pingSuccess = true

					// Verify MAC matches
					if normalizeMACAddress(hwAddr.String()) != normalizeMACAddress(h.MAC) {
						Warning("Host '%s' MAC mismatch - stored: %s, detected: %s at %s",
							h.Name, h.MAC, hwAddr.String(), hostIP)
					}
				} else {
					// Host in ARP table but not responding - flush stale entry for recovery
					FlushARPEntryIfExists(hostIP)
					Warning("Bulk ping - host '%s' not responding, flushed ARP cache for IP %s", h.Name, hostIP)
					pingSuccess = false
					arpSuccess = false
				}
			} else {
				// IP not in ARP table - do full network scan to find host by MAC
				_, pingOk, arpErr := ARPPingMAC(h.MAC, interfaceToUse, s.Config.PingTimeout)
				if arpErr == nil {
					// Active ARP scan succeeded
					pingSuccess = pingOk
					arpSuccess = true
				} else {
					// ARP methods failed - try static IP as fallback if configured
					if h.StaticIP != "" && h.UseAsFallback {
						hwAddr, fallbackErr := ARPPingIP(h.StaticIP, interfaceToUse)
						if fallbackErr == nil && hwAddr != nil {
							arpSuccess = true
							pingSuccess = true
						} else {
							// Both ARP and fallback failed
							if ipErr == nil && hostIP != "" {
								FlushARPEntryIfExists(hostIP)
							}
							s.PingCache.Set(cacheKey, false, false)
							result := map[string]interface{}{
								"host_id":      h.ID,
								"host_name":    h.Name,
								"ping_success": false,
								"arp_success":  false,
							}
							resultChan <- pingResult{index: idx, result: result}
							return
						}
					} else {
						// Both ARP methods failed - host is offline
						// Flush any potential stale ARP entry from earlier lookup
						if ipErr == nil && hostIP != "" {
							FlushARPEntryIfExists(hostIP)
							Debug("Bulk ping - both ARP methods failed for host '%s', flushed cache for IP %s", h.Name, hostIP)
						}
						s.PingCache.Set(cacheKey, false, false)
						result := map[string]interface{}{
							"host_id":      h.ID,
							"host_name":    h.Name,
							"ping_success": false,
							"arp_success":  false,
						}
						resultChan <- pingResult{index: idx, result: result}
						return
					}
				}
			}

			// Store result in cache
			s.PingCache.Set(cacheKey, pingSuccess, arpSuccess)

			// Frontend response (no sensitive data - no MAC, no IP)
			result := map[string]interface{}{
				"host_id":      h.ID,
				"host_name":    h.Name,
				"ping_success": pingSuccess,
				"arp_success":  arpSuccess,
			}

			resultChan <- pingResult{index: idx, result: result}
		}(i, host)
	}

	// Collect and stream results as they arrive
	// Keep track of which results we've received to maintain order awareness
	receivedCount := 0
	first := true

	for receivedCount < len(hosts) {
		res := <-resultChan
		receivedCount++

		if !first {
			w.Write([]byte(","))
		}
		first = false

		encoder.Encode(res.result)
		flusher.Flush()
	}

	// Close response array
	w.Write([]byte("]"))
}
