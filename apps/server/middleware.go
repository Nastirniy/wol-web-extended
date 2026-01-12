package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

// checkAuth verifies that the user is authenticated (if auth is enabled)
func (s *Server) checkAuth(w http.ResponseWriter, r *http.Request) bool {
	if !s.Config.UseAuth {
		return true
	}

	session, err := s.getSessionFromRequest(r)
	if err != nil {
		sendJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}

	// Check if session is still valid
	if session.Expires.Before(time.Now()) {
		s.deleteSession(session.ID)
		sendJSONError(w, "Session expired", http.StatusUnauthorized)
		return false
	}

	return true
}

// getCurrentUser returns the current user from the request session
func (s *Server) getCurrentUser(r *http.Request) *User {
	if !s.Config.UseAuth {
		return nil
	}

	session, err := s.getSessionFromRequest(r)
	if err != nil {
		return nil
	}

	// Check if session is still valid
	if session.Expires.Before(time.Now()) {
		s.deleteSession(session.ID)
		return nil
	}

	var user User
	err = s.DB.QueryRow("SELECT id, name, password, readonly, is_superuser, created, updated FROM users WHERE id = ?",
		session.UserID).Scan(&user.ID, &user.Name, &user.Password, &user.ReadOnly, &user.IsSuperuser, &user.Created, &user.Updated)

	if err != nil {
		return nil
	}

	return &user
}

// AuthMiddleware checks authentication and injects user into request context
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.checkAuth(w, r) {
			return
		}
		user := s.getCurrentUser(r)
		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserFromContext retrieves user from request context
func GetUserFromContext(r *http.Request) *User {
	user, _ := r.Context().Value("user").(*User)
	return user
}

// generateID generates a random ID for hosts, users, or sessions
func generateID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate ID: %w", err)
	}
	return hex.EncodeToString(bytes)[:15], nil
}
