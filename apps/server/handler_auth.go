package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// Authentication handlers

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.Config.UseAuth {
		sendJSONError(w, "Authentication not enabled", http.StatusBadRequest)
		return
	}

	var loginReq LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		sendJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Trim whitespace from credentials
	loginReq.Username = strings.TrimSpace(loginReq.Username)
	loginReq.Password = strings.TrimSpace(loginReq.Password)

	user, err := s.authenticateUser(loginReq.Username, loginReq.Password)
	if err != nil {
		sendJSONError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	session, err := s.createSession(user.ID)
	if err != nil {
		sendJSONError(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set secure HTTP-only cookie
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    session.ID,
		Expires:  session.Expires,
		HttpOnly: true,
		Secure:   s.Config.BehindProxy, // Secure when behind reverse proxy (Nginx, Caddy, etc.)
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}
	http.SetCookie(w, cookie)

	response := map[string]interface{}{
		"success": true,
		"user": map[string]interface{}{
			"id":   user.ID,
			"name": user.Name,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if !s.Config.UseAuth {
		sendJSONError(w, "Authentication not enabled", http.StatusBadRequest)
		return
	}

	session, err := s.getSessionFromRequest(r)
	if err == nil {
		s.deleteSession(session.ID)
	}

	// Clear the session cookie
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
	}
	http.SetCookie(w, cookie)

	response := map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	if !s.Config.UseAuth {
		response := map[string]interface{}{
			"authenticated": false,
			"auth_enabled":  false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	user := s.getCurrentUser(r)
	if user == nil {
		response := map[string]interface{}{
			"authenticated": false,
			"auth_enabled":  true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"authenticated": true,
		"auth_enabled":  true,
		"user": map[string]interface{}{
			"id":           user.ID,
			"name":         user.Name,
			"readonly":     user.ReadOnly,
			"is_superuser": user.IsSuperuser,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Initial setup handler - creates first superuser
func (s *Server) handleInitialSetup(w http.ResponseWriter, r *http.Request) {
	if !s.Config.UseAuth {
		sendJSONError(w, "Authentication not enabled", http.StatusBadRequest)
		return
	}

	// Check if any superuser exists
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM users WHERE is_superuser = TRUE").Scan(&count)
	if err != nil {
		sendJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}

	if count > 0 {
		sendJSONError(w, "Superuser already exists", http.StatusForbidden)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Trim whitespace from credentials
	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Password == "" {
		sendJSONError(w, "Username and password required", http.StatusBadRequest)
		return
	}

	// Create superuser
	userID, err := generateID()
	if err != nil {
		Error("Failed to generate user ID: %v", err)
		sendJSONError(w, "Failed to create superuser", http.StatusInternalServerError)
		return
	}
	hashedPassword := hashPassword(req.Password)

	_, err = s.DB.Exec(`INSERT INTO users (id, name, password, readonly, is_superuser) VALUES (?, ?, ?, ?, ?)`,
		userID, req.Username, hashedPassword, false, true)

	if err != nil {
		sendJSONError(w, "Failed to create superuser", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Superuser created successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Check if superuser exists
func (s *Server) handleHasSuperuser(w http.ResponseWriter, r *http.Request) {
	if !s.Config.UseAuth {
		sendJSONError(w, "Authentication not enabled", http.StatusBadRequest)
		return
	}

	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM users WHERE is_superuser = TRUE").Scan(&count)
	if err != nil {
		Error("Failed to check superuser status: %v", err)
		sendJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"has_superuser": count > 0,
		"auth_enabled":  s.Config.UseAuth,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Check if user is superuser
func (s *Server) checkSuperuser(w http.ResponseWriter, r *http.Request) (*User, bool) {
	if !s.checkAuth(w, r) {
		return nil, false
	}

	user := s.getCurrentUser(r)
	if user == nil {
		sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}

	if !user.IsSuperuser {
		sendJSONError(w, "Forbidden: Superuser access required", http.StatusForbidden)
		return nil, false
	}

	return user, true
}

