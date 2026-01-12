package main

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents a JSON error response
type ErrorResponse struct {
	Error  string `json:"error"`
	Code   string `json:"code,omitempty"`
	Status string `json:"status,omitempty"`
}

// sendJSONError sends a consistent JSON error response
func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:  message,
		Status: http.StatusText(statusCode),
	})
}

// sendJSONErrorWithCode sends a JSON error response with an error code for frontend localization
func sendJSONErrorWithCode(w http.ResponseWriter, message string, code string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:  message,
		Code:   code,
		Status: http.StatusText(statusCode),
	})
}

// handleValidationError sends appropriate error response for validation errors
// If err is a ValidationError, it sends the error with its code
// Otherwise, it sends a generic error
func handleValidationError(w http.ResponseWriter, err error, statusCode int) {
	if valErr, ok := err.(*ValidationError); ok {
		sendJSONErrorWithCode(w, valErr.Message, valErr.Code, statusCode)
	} else {
		sendJSONError(w, err.Error(), statusCode)
	}
}

// sendJSONSuccess sends a JSON success response with a message
func sendJSONSuccess(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": message,
	})
}

// sendJSON sends a JSON response with the given data and status code
func sendJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// determineNetworkInterface determines which network interface to use for a host
// based on configuration settings and host-specific interface settings.
//
// Logic:
//  1. Per-host disabled (enable_per_host_interfaces: false):
//     - Use global network_interface if specified
//     - Otherwise use all interfaces (empty string)
//  2. Per-host enabled (enable_per_host_interfaces: true):
//     - Use host-specific interface if specified (unless readonly_mode)
//     - Otherwise use all interfaces (empty string)
//     - Note: Does NOT fallback to network_interface
func (s *Server) determineNetworkInterface(host Host) string {
	// Per-host interface selection disabled: use global network_interface
	if !s.Config.EnablePerHostInterfaces {
		// Return configured interface or empty (all interfaces)
		return s.Config.DefaultNetworkInterface
	}

	// Per-host interface selection enabled
	// In readonly mode, ignore host-specific interface
	if s.Config.ReadOnlyMode {
		return s.Config.DefaultNetworkInterface
	}

	// Use host-specific interface if specified
	if host.Interface != "" {
		return host.Interface
	}

	// Host has no specific interface: use all interfaces (not network_interface)
	return ""
}
