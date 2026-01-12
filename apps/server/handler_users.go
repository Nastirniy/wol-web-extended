package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.getUsersList(w, r)
	case "POST":
		s.createUser(w, r)
	}
}

func (s *Server) getUsersList(w http.ResponseWriter, r *http.Request) {
	if !s.Config.UseAuth {
		sendJSONError(w, "Authentication not enabled", http.StatusBadRequest)
		return
	}

	if _, ok := s.checkSuperuser(w, r); !ok {
		return
	}

	rows, err := s.DB.Query("SELECT id, name, readonly, is_superuser, created, updated FROM users ORDER BY created DESC")
	if err != nil {
		sendJSONError(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id, name string
		var readonly, isSuperuser bool
		var created, updated time.Time

		err := rows.Scan(&id, &name, &readonly, &isSuperuser, &created, &updated)
		if err != nil {
			continue
		}

		users = append(users, map[string]interface{}{
			"id":           id,
			"name":         name,
			"readonly":     readonly,
			"is_superuser": isSuperuser,
			"created":      created,
			"updated":      updated,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	if !s.Config.UseAuth {
		sendJSONError(w, "Authentication not enabled", http.StatusBadRequest)
		return
	}

	if _, ok := s.checkSuperuser(w, r); !ok {
		return
	}

	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		ReadOnly    bool   `json:"readonly"`
		IsSuperuser bool   `json:"is_superuser"`
	}

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

	// Check if username already exists
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM users WHERE name = ?", req.Username).Scan(&count)
	if err == nil && count > 0 {
		sendJSONError(w, "Username already exists", http.StatusConflict)
		return
	}

	userID, err := generateID()
	if err != nil {
		Error("Failed to generate user ID: %v", err)
		sendJSONError(w, "Failed to create user", http.StatusInternalServerError)
		return
	}
	hashedPassword := hashPassword(req.Password)

	_, err = s.DB.Exec(`INSERT INTO users (id, name, password, readonly, is_superuser) VALUES (?, ?, ?, ?, ?)`,
		userID, req.Username, hashedPassword, req.ReadOnly, req.IsSuperuser)

	if err != nil {
		sendJSONError(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "User created successfully",
		"user_id": userID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Handle individual user operations
func (s *Server) handleUserDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	switch r.Method {
	case "GET":
		s.getUser(w, r, userID)
	case "PUT":
		s.updateUser(w, r, userID)
	case "DELETE":
		s.deleteUser(w, r, userID)
	}
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request, userID string) {
	if !s.Config.UseAuth {
		sendJSONError(w, "Authentication not enabled", http.StatusBadRequest)
		return
	}

	if _, ok := s.checkSuperuser(w, r); !ok {
		return
	}

	var id, name string
	var readonly, isSuperuser bool
	var created, updated time.Time

	err := s.DB.QueryRow("SELECT id, name, readonly, is_superuser, created, updated FROM users WHERE id = ?", userID).
		Scan(&id, &name, &readonly, &isSuperuser, &created, &updated)

	if err == sql.ErrNoRows {
		sendJSONError(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		sendJSONError(w, "Failed to fetch user", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":           id,
		"name":         name,
		"readonly":     readonly,
		"is_superuser": isSuperuser,
		"created":      created,
		"updated":      updated,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request, userID string) {
	if !s.Config.UseAuth {
		sendJSONError(w, "Authentication not enabled", http.StatusBadRequest)
		return
	}

	currentUser, ok := s.checkSuperuser(w, r)
	if !ok {
		return
	}

	var req struct {
		Name        string `json:"name"`
		Password    string `json:"password"`
		ReadOnly    bool   `json:"readonly"`
		IsSuperuser bool   `json:"is_superuser"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Trim whitespace from credentials
	req.Name = strings.TrimSpace(req.Name)
	req.Password = strings.TrimSpace(req.Password)

	// Get current user data to check if they are a superuser
	var currentIsSuperuser bool
	err := s.DB.QueryRow("SELECT is_superuser FROM users WHERE id = ?", userID).Scan(&currentIsSuperuser)
	if err != nil {
		sendJSONError(w, "User not found", http.StatusNotFound)
		return
	}

	// Prevent password changes for superusers
	if currentIsSuperuser && req.Password != "" {
		sendJSONErrorWithCode(w, "Superuser passwords cannot be changed through the API. Use CLI to reset.", ErrCodeCannotChangeSuperuserPassword, http.StatusForbidden)
		return
	}

	// Prevent user from removing their own superuser status
	if currentUser.ID == userID && !req.IsSuperuser {
		sendJSONErrorWithCode(w, "Cannot remove your own superuser status", ErrCodeCannotRemoveOwnSuperuser, http.StatusForbidden)
		return
	}

	// Check if trying to remove superuser status from the last superuser
	// Only check this if the user IS currently a superuser AND we're trying to remove it
	if currentIsSuperuser && !req.IsSuperuser {
		var superuserCount int
		s.DB.QueryRow("SELECT COUNT(*) FROM users WHERE is_superuser = TRUE").Scan(&superuserCount)
		if superuserCount <= 1 {
			sendJSONErrorWithCode(w, "Cannot remove superuser status from the last superuser", ErrCodeCannotRemoveLastSuperuser, http.StatusForbidden)
			return
		}
	}

	var query string
	var args []interface{}

	if req.Password != "" {
		hashedPassword := hashPassword(req.Password)
		query = "UPDATE users SET name = ?, password = ?, readonly = ?, is_superuser = ?, updated = CURRENT_TIMESTAMP WHERE id = ?"
		args = []interface{}{req.Name, hashedPassword, req.ReadOnly, req.IsSuperuser, userID}
	} else {
		query = "UPDATE users SET name = ?, readonly = ?, is_superuser = ?, updated = CURRENT_TIMESTAMP WHERE id = ?"
		args = []interface{}{req.Name, req.ReadOnly, req.IsSuperuser, userID}
	}

	result, err := s.DB.Exec(query, args...)
	if err != nil {
		sendJSONError(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		sendJSONError(w, "User not found", http.StatusNotFound)
		return
	}

	// If password was changed, terminate all sessions for this user
	if req.Password != "" {
		_, err = s.DB.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
		if err != nil {
			Warning("Failed to delete sessions for user %s: %v", userID, err)
			// Don't fail the request, just log the error
		} else {
			Info("Terminated all sessions for user %s after password change", userID)
		}
	}

	response := map[string]interface{}{
		"success": true,
		"message": "User updated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request, userID string) {
	if !s.Config.UseAuth {
		sendJSONError(w, "Authentication not enabled", http.StatusBadRequest)
		return
	}

	currentUser, ok := s.checkSuperuser(w, r)
	if !ok {
		return
	}

	// Prevent user from deleting themselves
	if currentUser.ID == userID {
		sendJSONErrorWithCode(w, "Cannot delete your own account", ErrCodeCannotDeleteSelf, http.StatusForbidden)
		return
	}

	// Check if this is a superuser being deleted
	var isSuperuser bool
	err := s.DB.QueryRow("SELECT is_superuser FROM users WHERE id = ?", userID).Scan(&isSuperuser)
	if err == sql.ErrNoRows {
		sendJSONError(w, "User not found", http.StatusNotFound)
		return
	}

	// If deleting a superuser, make sure it's not the last one
	if isSuperuser {
		var superuserCount int
		s.DB.QueryRow("SELECT COUNT(*) FROM users WHERE is_superuser = TRUE").Scan(&superuserCount)
		if superuserCount <= 1 {
			sendJSONErrorWithCode(w, "Cannot delete the last superuser", ErrCodeCannotDeleteLastSuperuser, http.StatusForbidden)
			return
		}
	}

	result, err := s.DB.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		sendJSONError(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		sendJSONError(w, "User not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
