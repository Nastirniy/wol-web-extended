package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

type Session struct {
	ID      string    `json:"id"`
	UserID  string    `json:"user_id"`
	Expires time.Time `json:"expires"`
	Created time.Time `json:"created"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Session management functions
func (s *Server) createSession(userID string) (*Session, error) {
	sessionID, err := generateSecureID()
	if err != nil {
		Error("Failed to generate session ID: %v", err)
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	expires := time.Now().Add(time.Duration(s.Config.AuthExpireHours * float64(time.Hour)))

	Debug("Creating session: AuthExpireHours=%.4f, Duration=%v, Expires at %v",
		s.Config.AuthExpireHours,
		time.Duration(s.Config.AuthExpireHours*float64(time.Hour)),
		expires)

	session := &Session{
		ID:      sessionID,
		UserID:  userID,
		Expires: expires,
		Created: time.Now(),
	}

	_, err = s.DB.Exec("INSERT OR REPLACE INTO sessions (id, user_id, expires, created) VALUES (?, ?, ?, ?)",
		session.ID, session.UserID, session.Expires, session.Created)

	if err != nil {
		Error("Failed to insert session into database: %v", err)
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

func (s *Server) getSessionFromRequest(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil, err
	}

	return s.getSession(cookie.Value)
}

func (s *Server) getSession(sessionID string) (*Session, error) {
	var session Session
	now := time.Now()
	err := s.DB.QueryRow("SELECT id, user_id, expires, created FROM sessions WHERE id = ? AND expires > ?",
		sessionID, now).Scan(&session.ID, &session.UserID, &session.Expires, &session.Created)

	if err != nil {
		Debug("Session lookup failed for ID=%s, now=%v, error=%v", sessionID, now, err)
		return nil, err
	}

	return &session, nil
}

func (s *Server) deleteSession(sessionID string) error {
	_, err := s.DB.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	return err
}

func (s *Server) cleanupExpiredSessions() error {
	_, err := s.DB.Exec("DELETE FROM sessions WHERE expires <= ?", time.Now())
	return err
}

func (s *Server) authenticateUser(username, password string) (*User, error) {
	var user User

	// First, get the user by username
	err := s.DB.QueryRow("SELECT id, name, password, readonly, is_superuser, created, updated FROM users WHERE name = ?",
		username).Scan(&user.ID, &user.Name, &user.Password, &user.ReadOnly, &user.IsSuperuser, &user.Created, &user.Updated)

	if err != nil {
		return nil, err
	}

	// Verify password using bcrypt
	if !verifyPassword(user.Password, password) {
		return nil, sql.ErrNoRows // Return same error as "user not found" to prevent user enumeration
	}

	return &user, nil
}

func generateSecureID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure ID: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
