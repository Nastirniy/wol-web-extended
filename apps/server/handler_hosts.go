package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// handleHosts handles GET (list) and POST (create) for hosts
func (s *Server) handleHosts(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)

	switch r.Method {
	case "GET":
		s.getHosts(w, r, user)
	case "POST":
		s.createHost(w, r, user)
	}
}

// handleHost handles GET, PUT, DELETE for a specific host
func (s *Server) handleHost(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	vars := mux.Vars(r)
	hostID := vars["id"]

	switch r.Method {
	case "GET":
		s.getHost(w, r, user, hostID)
	case "PUT":
		s.updateHost(w, r, user, hostID)
	case "DELETE":
		s.deleteHost(w, r, user, hostID)
	}
}

// getHosts returns all hosts for the current user
func (s *Server) getHosts(w http.ResponseWriter, r *http.Request, user *User) {
	var rows *sql.Rows
	var err error

	userDesc := "anonymous"
	if user != nil {
		userDesc = user.ID
	}
	Debug("Fetching hosts for user: %s (auth mode: %v)", userDesc, s.Config.UseAuth)

	if s.Config.UseAuth && user != nil {
		// In auth mode, only show user's own hosts
		rows, err = s.DB.Query("SELECT id, name, mac, broadcast, interface, static_ip, use_as_fallback, user_id, created, updated FROM hosts WHERE user_id = ? ORDER BY created DESC", user.ID)
	} else {
		// In no-auth mode, only show hosts created in no-auth mode (NULL user_id)
		rows, err = s.DB.Query("SELECT id, name, mac, broadcast, interface, static_ip, use_as_fallback, user_id, created, updated FROM hosts WHERE user_id IS NULL ORDER BY created DESC")
	}

	if err != nil {
		Debug("Failed to fetch hosts for user %s: %v", userDesc, err)
		sendJSONError(w, "Failed to fetch hosts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var hosts []Host
	for rows.Next() {
		var host Host
		err := rows.Scan(&host.ID, &host.Name, &host.MAC, &host.Broadcast, &host.Interface, &host.StaticIP, &host.UseAsFallback, &host.UserID, &host.Created, &host.Updated)
		if err != nil {
			continue
		}

		// Apply readonly restrictions for sensitive data
		if (s.Config.UseAuth && user != nil && user.ReadOnly) || s.Config.ReadOnlyMode {
			host.MAC = ""
			host.Broadcast = ""
			host.Interface = ""
			host.StaticIP = ""
			host.UseAsFallback = false
		}

		// Hide interface data when per-host interface selection is disabled
		if !s.Config.EnablePerHostInterfaces {
			host.Interface = ""
		}

		hosts = append(hosts, host)
	}

	Debug("Returning %d hosts for user: %s", len(hosts), userDesc)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hosts)
}

// createHost creates a new host
func (s *Server) createHost(w http.ResponseWriter, r *http.Request, user *User) {
	userDesc := "anonymous"
	if user != nil {
		userDesc = user.ID
	}
	Debug("Create host request from user: %s", userDesc)

	// Check if modifications are allowed
	if s.Config.ReadOnlyMode || (s.Config.UseAuth && user != nil && user.ReadOnly) {
		Debug("Create host denied for user %s (readonly mode)", userDesc)
		sendJSONError(w, "Read-only access: modification not allowed", http.StatusForbidden)
		return
	}

	var host Host
	if err := json.NewDecoder(r.Body).Decode(&host); err != nil {
		Debug("Failed to decode create host request from user %s: %v", userDesc, err)
		sendJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	Debug("Creating host '%s' (MAC: %s, Broadcast: %s) for user: %s",
		host.Name, host.MAC, host.Broadcast, userDesc)

	// Ignore any user_id sent from client - we'll set it based on auth state
	host.UserID = nil

	// Trim whitespace from all string fields
	host.Name = strings.TrimSpace(host.Name)
	host.MAC = strings.TrimSpace(host.MAC)
	host.Broadcast = strings.TrimSpace(host.Broadcast)
	host.Interface = strings.TrimSpace(host.Interface)
	host.StaticIP = strings.TrimSpace(host.StaticIP)

	if err := sanitizeHostName(host.Name); err != nil {
		handleValidationError(w, err, http.StatusBadRequest)
		return
	}

	if err := sanitizeMACAddress(host.MAC); err != nil {
		handleValidationError(w, err, http.StatusBadRequest)
		return
	}
	host.MAC = normalizeMACAddress(host.MAC)

	if err := sanitizeBroadcastAddress(host.Broadcast); err != nil {
		handleValidationError(w, err, http.StatusBadRequest)
		return
	}

	// Validate static IP if provided
	if err := sanitizeStaticIPv4(host.StaticIP); err != nil {
		handleValidationError(w, err, http.StatusBadRequest)
		return
	}

	// Reject interface specification when per-host interface selection is disabled
	if host.Interface != "" && !s.Config.EnablePerHostInterfaces {
		sendJSONErrorWithCode(w, "Per-host network interface selection is disabled. Remove the interface field.", ErrCodeForbidden, http.StatusBadRequest)
		return
	}

	if host.Interface != "" {
		if err := validateNetworkInterface(host.Interface); err != nil {
			sendJSONError(w, "Invalid network interface: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	hostID, err := generateID()
	if err != nil {
		Error("Failed to generate host ID: %v", err)
		sendJSONError(w, "Failed to generate host ID", http.StatusInternalServerError)
		return
	}
	host.ID = hostID

	// Validate authentication requirements
	if s.Config.UseAuth {
		// In auth mode, user must be authenticated
		if user == nil {
			sendJSONError(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		host.UserID = &user.ID
	} else {
		// In no-auth mode, always set user_id to NULL
		host.UserID = nil
	}

	_, err = s.DB.Exec("INSERT INTO hosts (id, name, mac, broadcast, interface, static_ip, use_as_fallback, user_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		host.ID, host.Name, host.MAC, host.Broadcast, host.Interface, host.StaticIP, host.UseAsFallback, host.UserID)

	if err != nil {
		Debug("Failed to create host '%s' for user %s: %v", host.Name, userDesc, err)
		sendJSONError(w, "Failed to create host", http.StatusInternalServerError)
		return
	}

	Debug("Host '%s' (ID: %s, MAC: %s) created successfully for user: %s",
		host.Name, host.ID, host.MAC, userDesc)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(host)
}

// getHost returns a specific host by ID
func (s *Server) getHost(w http.ResponseWriter, r *http.Request, user *User, hostID string) {
	var host Host
	var query string
	var args []interface{}

	if s.Config.UseAuth && user != nil {
		query = "SELECT id, name, mac, broadcast, interface, static_ip, use_as_fallback, user_id, created, updated FROM hosts WHERE id = ? AND user_id = ?"
		args = []interface{}{hostID, user.ID}
	} else {
		// In no-auth mode, ONLY allow access to hosts with NULL user_id
		query = "SELECT id, name, mac, broadcast, interface, static_ip, use_as_fallback, user_id, created, updated FROM hosts WHERE id = ? AND user_id IS NULL"
		args = []interface{}{hostID}
	}

	err := s.DB.QueryRow(query, args...).Scan(&host.ID, &host.Name, &host.MAC, &host.Broadcast, &host.Interface, &host.StaticIP, &host.UseAsFallback, &host.UserID, &host.Created, &host.Updated)
	if err == sql.ErrNoRows {
		// In no-auth mode, this could mean the host doesn't exist OR it belongs to a user
		if !s.Config.UseAuth {
			// Check if host exists but has a user_id (security violation attempt)
			var exists bool
			err := s.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM hosts WHERE id = ?)", hostID).Scan(&exists)
			if err == nil && exists {
				sendJSONError(w, "Access denied: host not accessible in no-auth mode", http.StatusForbidden)
				return
			}
		}
		sendJSONError(w, "Host not found", http.StatusNotFound)
		return
	}
	if err != nil {
		sendJSONError(w, "Failed to fetch host", http.StatusInternalServerError)
		return
	}

	// Apply readonly restrictions for sensitive data
	if (s.Config.UseAuth && user != nil && user.ReadOnly) || s.Config.ReadOnlyMode {
		host.MAC = ""
		host.Interface = ""
		host.StaticIP = ""
		host.UseAsFallback = false
	}

	// Hide interface data when per-host interface selection is disabled
	if !s.Config.EnablePerHostInterfaces {
		host.Interface = ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(host)
}

// updateHost updates an existing host
func (s *Server) updateHost(w http.ResponseWriter, r *http.Request, user *User, hostID string) {
	userDesc := "anonymous"
	if user != nil {
		userDesc = user.ID
	}
	Debug("Update host request for ID %s from user: %s", hostID, userDesc)

	// Check if modifications are allowed
	if s.Config.ReadOnlyMode || (s.Config.UseAuth && user != nil && user.ReadOnly) {
		Debug("Update host denied for user %s (readonly mode)", userDesc)
		sendJSONError(w, "Read-only access: modification not allowed", http.StatusForbidden)
		return
	}

	var host Host
	if err := json.NewDecoder(r.Body).Decode(&host); err != nil {
		sendJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Trim whitespace from all string fields
	host.Name = strings.TrimSpace(host.Name)
	host.MAC = strings.TrimSpace(host.MAC)
	host.Broadcast = strings.TrimSpace(host.Broadcast)
	host.Interface = strings.TrimSpace(host.Interface)
	host.StaticIP = strings.TrimSpace(host.StaticIP)

	if err := sanitizeHostName(host.Name); err != nil {
		handleValidationError(w, err, http.StatusBadRequest)
		return
	}

	if err := sanitizeMACAddress(host.MAC); err != nil {
		handleValidationError(w, err, http.StatusBadRequest)
		return
	}
	host.MAC = normalizeMACAddress(host.MAC)

	if err := sanitizeBroadcastAddress(host.Broadcast); err != nil {
		handleValidationError(w, err, http.StatusBadRequest)
		return
	}

	// Validate static IP if provided
	if err := sanitizeStaticIPv4(host.StaticIP); err != nil {
		handleValidationError(w, err, http.StatusBadRequest)
		return
	}

	// Reject interface specification when per-host interface selection is disabled
	if host.Interface != "" && !s.Config.EnablePerHostInterfaces {
		sendJSONErrorWithCode(w, "Per-host network interface selection is disabled. Remove the interface field.", ErrCodeForbidden, http.StatusBadRequest)
		return
	}

	if host.Interface != "" {
		if err := validateNetworkInterface(host.Interface); err != nil {
			sendJSONError(w, "Invalid network interface: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	var query string
	var args []interface{}

	if s.Config.UseAuth && user != nil {
		query = "UPDATE hosts SET name = ?, mac = ?, broadcast = ?, interface = ?, static_ip = ?, use_as_fallback = ?, updated = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?"
		args = []interface{}{host.Name, host.MAC, host.Broadcast, host.Interface, host.StaticIP, host.UseAsFallback, hostID, user.ID}
	} else {
		query = "UPDATE hosts SET name = ?, mac = ?, broadcast = ?, interface = ?, static_ip = ?, use_as_fallback = ?, updated = CURRENT_TIMESTAMP WHERE id = ? AND user_id IS NULL"
		args = []interface{}{host.Name, host.MAC, host.Broadcast, host.Interface, host.StaticIP, host.UseAsFallback, hostID}
	}

	result, err := s.DB.Exec(query, args...)
	if err != nil {
		Debug("Failed to update host ID %s for user %s: %v", hostID, userDesc, err)
		sendJSONError(w, "Failed to update host", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// In no-auth mode, check if host exists but has a user_id
		if !s.Config.UseAuth {
			var exists bool
			err := s.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM hosts WHERE id = ?)", hostID).Scan(&exists)
			if err == nil && exists {
				Debug("Update host denied - ID %s belongs to user in no-auth mode", hostID)
				sendJSONError(w, "Access denied: host not accessible in no-auth mode", http.StatusForbidden)
				return
			}
		}
		Debug("Update host failed - ID %s not found or unauthorized for user %s", hostID, userDesc)
		sendJSONError(w, "Host not found", http.StatusNotFound)
		return
	}

	Debug("Host '%s' (ID: %s, MAC: %s) updated successfully by user: %s",
		host.Name, hostID, host.MAC, userDesc)

	host.ID = hostID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(host)
}

// deleteHost deletes a host
func (s *Server) deleteHost(w http.ResponseWriter, r *http.Request, user *User, hostID string) {
	userDesc := "anonymous"
	if user != nil {
		userDesc = user.ID
	}
	Debug("Delete host request for ID %s from user: %s", hostID, userDesc)

	// Check if modifications are allowed
	if s.Config.ReadOnlyMode || (s.Config.UseAuth && user != nil && user.ReadOnly) {
		Debug("Delete host denied for user %s (readonly mode)", userDesc)
		sendJSONError(w, "Read-only access: modification not allowed", http.StatusForbidden)
		return
	}

	var query string
	var args []interface{}

	if s.Config.UseAuth && user != nil {
		query = "DELETE FROM hosts WHERE id = ? AND user_id = ?"
		args = []interface{}{hostID, user.ID}
	} else {
		query = "DELETE FROM hosts WHERE id = ? AND user_id IS NULL"
		args = []interface{}{hostID}
	}

	result, err := s.DB.Exec(query, args...)
	if err != nil {
		Debug("Failed to delete host ID %s for user %s: %v", hostID, userDesc, err)
		sendJSONError(w, "Failed to delete host", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// In no-auth mode, check if host exists but has a user_id
		if !s.Config.UseAuth {
			var exists bool
			err := s.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM hosts WHERE id = ?)", hostID).Scan(&exists)
			if err == nil && exists {
				Debug("Delete host denied - ID %s belongs to user in no-auth mode", hostID)
				sendJSONError(w, "Access denied: host not accessible in no-auth mode", http.StatusForbidden)
				return
			}
		}
		Debug("Delete host failed - ID %s not found or unauthorized for user %s", hostID, userDesc)
		sendJSONError(w, "Host not found", http.StatusNotFound)
		return
	}

	Debug("Host ID %s deleted successfully by user: %s", hostID, userDesc)
	w.WriteHeader(http.StatusNoContent)
}
