package main

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// buildURL creates a URL path with the configured URL prefix
func (s *Server) buildURL(path string) string {
	prefix := s.Config.URLPrefix
	if prefix == "" {
		return path
	}

	// Normalize prefix
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if strings.HasSuffix(prefix, "/") {
		prefix = strings.TrimSuffix(prefix, "/")
	}

	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return prefix + path
}

// setupRoutes configures all HTTP routes for the application
func (s *Server) setupRoutes(router *mux.Router) {
	apiPrefix := s.Config.URLPrefix
	if apiPrefix != "" && !strings.HasPrefix(apiPrefix, "/") {
		apiPrefix = "/" + apiPrefix
	}
	if apiPrefix != "" && strings.HasSuffix(apiPrefix, "/") {
		apiPrefix = strings.TrimSuffix(apiPrefix, "/")
	}

	api := router.PathPrefix(apiPrefix + "/api").Subrouter()

	// Health check endpoint (no authentication required)
	api.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Authentication endpoints (no auth middleware needed)
	api.HandleFunc("/auth/login", s.handleLogin).Methods("POST")
	api.HandleFunc("/auth/logout", s.handleLogout).Methods("POST")
	api.HandleFunc("/auth/me", s.handleAuthMe).Methods("GET")
	api.HandleFunc("/auth/setup", s.handleInitialSetup).Methods("POST")
	api.HandleFunc("/auth/has-superuser", s.handleHasSuperuser).Methods("GET")

	// Protected endpoints - apply auth middleware
	protected := api.PathPrefix("").Subrouter()
	protected.Use(s.AuthMiddleware)

	// Configuration endpoints
	protected.HandleFunc("/config", s.handleConfig).Methods("GET")
	protected.HandleFunc("/network-interfaces", s.handleNetworkInterfaces).Methods("GET")

	// Host management endpoints
	protected.HandleFunc("/hosts", s.handleHosts).Methods("GET", "POST")
	protected.HandleFunc("/hosts/{id}", s.handleHost).Methods("GET", "PUT", "DELETE")

	// Ping endpoints
	protected.HandleFunc("/ping", s.handlePing).Methods("POST")
	protected.HandleFunc("/ping/bulk", s.handleBulkPing).Methods("POST")

	// Wake-on-LAN endpoint
	protected.HandleFunc("/wake", s.handleWake).Methods("POST")

	// User management endpoints (superuser only)
	protected.HandleFunc("/users", s.handleUsers).Methods("GET", "POST")
	protected.HandleFunc("/users/{id}", s.handleUserDetail).Methods("GET", "PUT", "DELETE")

	// Setup static file serving with SPA routing support
	if apiPrefix != "" {
		// With prefix: serve static files at the prefix root and catch-all
		router.PathPrefix(apiPrefix + "/").Handler(http.StripPrefix(apiPrefix, s.spaHandler("./static/")))
	} else {
		// Without prefix: serve static files at root
		router.PathPrefix("/").Handler(s.spaHandler("./static/"))
	}
}
