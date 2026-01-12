package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleWake(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)

	// Wake-on-LAN is always allowed regardless of readonly mode
	// Readonly mode only restricts creating/modifying/deleting hosts

	userKey := "anonymous"
	if user != nil {
		userKey = user.ID
		Debug("WoL request from user: %s", userKey)
	} else {
		Debug("WoL request from anonymous user")
	}

	if !s.WoLRateLimit.Allow(userKey) {
		Debug("WoL rate limit exceeded for user: %s", userKey)
		response := map[string]string{
			"message": "Rate limit exceeded",
			"error":   "Please wait before sending more Wake-on-LAN requests",
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
		Debug("Failed to decode WoL request body: %v", err)
		sendJSONError(w, "Failed to read request data", http.StatusBadRequest)
		return
	}

	Debug("WoL request for host ID: %s", data.ID)

	var host Host
	var query string
	var args []interface{}

	if s.Config.UseAuth && user != nil {
		query = "SELECT id, name, mac, broadcast, interface, user_id FROM hosts WHERE id = ? AND user_id = ?"
		args = []interface{}{data.ID, user.ID}
	} else {
		// In no-auth mode, ONLY allow access to hosts with NULL user_id
		query = "SELECT id, name, mac, broadcast, interface, user_id FROM hosts WHERE id = ? AND user_id IS NULL"
		args = []interface{}{data.ID}
	}

	err := s.DB.QueryRow(query, args...).Scan(&host.ID, &host.Name, &host.MAC, &host.Broadcast, &host.Interface, &host.UserID)
	if err == sql.ErrNoRows {
		// In no-auth mode, check if host exists but has a user_id
		if !s.Config.UseAuth {
			var exists bool
			err := s.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM hosts WHERE id = ?)", data.ID).Scan(&exists)
			if err == nil && exists {
				Debug("WoL access denied for host ID %s (belongs to user in no-auth mode)", data.ID)
				sendJSONError(w, "Access denied: host not accessible in no-auth mode", http.StatusForbidden)
				return
			}
		}
		Debug("WoL failed - host ID %s not found", data.ID)
		sendJSONError(w, "Host not found", http.StatusNotFound)
		return
	}
	if err != nil {
		Debug("WoL failed - database error for host ID %s: %v", data.ID, err)
		sendJSONError(w, "Failed to find host", http.StatusInternalServerError)
		return
	}

	Debug("Found host '%s' (ID: %s, MAC: %s, Broadcast: %s)",
		host.Name, host.ID, host.MAC, host.Broadcast)

	parts := strings.Split(host.Broadcast, ":")
	if len(parts) != 2 {
		Debug("WoL failed - invalid broadcast format for host '%s': %s", host.Name, host.Broadcast)
		sendJSONError(w, "Invalid broadcast format", http.StatusBadRequest)
		return
	}

	targetIp := parts[0]
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		Debug("WoL failed - invalid port for host '%s': %s", host.Name, parts[1])
		sendJSONError(w, "Invalid port in broadcast field", http.StatusBadRequest)
		return
	}

	// Determine which network interface(s) to use for WoL
	interfaceToUse := s.determineNetworkInterface(host)

	ifaceDesc := "all available interfaces"
	if interfaceToUse != "" {
		ifaceDesc = interfaceToUse
	}

	Debug("Sending WoL magic packet for host '%s' (MAC: %s) to %s:%d using %s",
		host.Name, host.MAC, targetIp, port, ifaceDesc)

	err = SendWakeOnLanWithInterface(host.MAC, targetIp, port, interfaceToUse)
	if err != nil {
		Debug("WoL packet send FAILED for host '%s' (MAC: %s) to %s:%d - %v",
			host.Name, host.MAC, targetIp, port, err)
		response := map[string]string{
			"message": "Failed to wake host",
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Record WoL usage for prioritization in ping queue
	s.WoLHistory.RecordWoL(host.ID)
	Debug("Recorded WoL event for host '%s' (ID: %s) in priority queue", host.Name, host.ID)

	// Invalidate ping cache for this host (status will change after WoL)
	s.PingCache.Invalidate(host.ID)
	Debug("Invalidated ping cache for host '%s' (ID: %s) after WoL", host.Name, host.ID)

	Debug("WoL magic packet SUCCESSFULLY sent for host '%s' (MAC: %s) to %s:%d",
		host.Name, host.MAC, targetIp, port)
	response := map[string]string{"message": "WakeOnLan Magic Packet Sent"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
