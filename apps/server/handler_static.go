package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Static asset extensions
var staticAssetExtensions = map[string]bool{
	".css":   true,
	".js":    true,
	".png":   true,
	".jpg":   true,
	".jpeg":  true,
	".gif":   true,
	".webp":  true,
	".svg":   true,
	".ico":   true,
	".woff":  true,
	".woff2": true,
	".ttf":   true,
	".eot":   true,
	".otf":   true,
	".map":   true,
}

// SPA route mappings
var spaRoutes = map[string]string{
	"/auth":  "/auth.html",
	"/users": "/index.html", // Users page is part of the main SPA
}

// Valid SPA routes that should serve index.html
var validSPARoutes = map[string]bool{
	"/":      true,
	"/home":  true,
	"/auth":  true,
	"/users": true,
	"/setup": true,
}

// Public routes that don't require authentication
var publicRoutes = map[string]bool{
	"/auth":                   true,
	"/api/config":             true, // Config endpoint is public (doesn't reveal sensitive data)
	"/api/auth/setup":         true, // Setup endpoint for creating first superuser
	"/api/auth/has-superuser": true, // Check if superuser exists
	"/api/auth/login":         true, // Login endpoint
	"/api/auth/me":            true, // Auth status check (returns not authenticated if no session)
}

// spaHandler serves static files and handles SPA routing with authentication
func (s *Server) spaHandler(staticPath string) http.Handler {
	// Pre-compile file server for better performance
	fileServer := http.FileServer(http.Dir(staticPath))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security: Prevent directory traversal attacks
		if strings.Contains(r.URL.Path, "..") {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		// Only allow GET and HEAD methods for static files
		if r.Method != "GET" && r.Method != "HEAD" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Use forward slashes for consistency (filepath.Clean uses OS separators on Windows)
		path := strings.ReplaceAll(filepath.Clean(r.URL.Path), "\\", "/")

		// Check if this is a static asset by file extension
		ext := filepath.Ext(path)
		isStaticAsset := staticAssetExtensions[ext]

		// Check if this is a public route
		isPublicRoute := publicRoutes[path]
		if !isPublicRoute {
			// Check if path starts with any public route
			for route := range publicRoutes {
				if strings.HasPrefix(path, route) {
					isPublicRoute = true
					break
				}
			}
		}

		// If user is authenticated and trying to access /auth, redirect to home
		if s.Config.UseAuth && path == "/auth" && s.isAuthenticated(r) {
			http.Redirect(w, r, s.buildURL("/home"), http.StatusSeeOther)
			return
		}

		// Check if physical file exists
		filePath := filepath.Join(staticPath, path)
		fileInfo, err := os.Stat(filePath)

		if err == nil && !fileInfo.IsDir() {
			// File exists - set appropriate cache headers
			s.setStaticCacheHeaders(w, ext, isStaticAsset)
			fileServer.ServeHTTP(w, r)
			return
		}

		// File doesn't exist - determine what to do based on path type
		// If this is clearly a static asset (has extension or build directories), return 404
		if isStaticAsset || strings.Contains(path, "/_app/") || strings.Contains(path, "/.vite/") {
			http.NotFound(w, r)
			return
		}

		// SPA routing: Map routes to HTML files (these support prefixes)
		for routePrefix, htmlFile := range spaRoutes {
			if path == routePrefix {
				// Check auth before serving valid route
				if s.Config.UseAuth && !publicRoutes[path] && !s.isAuthenticated(r) {
					s.redirectToAuthOrSetup(w, r)
					return
				}
				s.serveSPAPage(w, r, filepath.Join(staticPath, htmlFile))
				return
			}
		}

		// Check if this is an exact match for a valid SPA route
		if validSPARoutes[path] {
			// Check auth before serving valid route
			if s.Config.UseAuth && !publicRoutes[path] && !s.isAuthenticated(r) {
				s.redirectToAuthOrSetup(w, r)
				return
			}
			// Valid SPA route - serve index.html
			s.serveSPAPage(w, r, filepath.Join(staticPath, "index.html"))
			return
		}

		// Not a valid route - serve 404 error page (NO auth check - always show 404)
		s.serve404Page(w, r, staticPath)
	})
}

// isAuthenticated checks if the request has a valid session
func (s *Server) isAuthenticated(r *http.Request) bool {
	session, err := s.getSessionFromRequest(r)
	return err == nil && session != nil
}

// redirectToAuthOrSetup redirects to auth page
func (s *Server) redirectToAuthOrSetup(w http.ResponseWriter, r *http.Request) {
	// Don't redirect if already on auth page (prevents redirect loop)
	currentPath := strings.ReplaceAll(filepath.Clean(r.URL.Path), "\\", "/")
	if currentPath == "/auth" {
		return // Let the request continue to serve auth.html
	}

	// Redirect to auth page
	// If no superuser exists, auth page will show setup UI
	http.Redirect(w, r, s.buildURL("/auth"), http.StatusSeeOther)
}

// setStaticCacheHeaders sets appropriate cache headers for static assets
func (s *Server) setStaticCacheHeaders(w http.ResponseWriter, ext string, isStaticAsset bool) {
	// Security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	if isStaticAsset {
		// Cache static assets for 1 year (immutable)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		// Don't cache HTML files (always revalidate)
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}

	// Set appropriate Content-Type
	switch ext {
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	case ".woff":
		w.Header().Set("Content-Type", "font/woff")
	case ".woff2":
		w.Header().Set("Content-Type", "font/woff2")
	}
}

// injectBaseHref injects <base href> as the FIRST element in <head> and fixes absolute paths.
//
// This function is critical for supporting URL prefixes (e.g., /wolweb) when running behind
// a reverse proxy. It performs the following operations:
//
//  1. Injects <base href="/prefix/"> as the first element in <head>
//     - This allows SvelteKit to correctly resolve relative paths
//     - Must be first to take precedence over other head elements
//
//  2. Converts absolute paths to relative paths:
//     - href="/_app/..." -> href="_app/..." (removes leading slash)
//     - href="/favicon..." -> href="favicon..." (removes leading slash)
//     - src="/_app/..." -> src="_app/..." (removes leading slash)
//     - This makes the <base> tag work correctly for all resources
//
//  3. Fixes JavaScript module imports:
//     - import("/_app/...") -> import("./_app/...")
//     - Ensures dynamic imports work with base href
//
//  4. Updates SvelteKit base configuration:
//     - base: "" -> base: "/wolweb" (or configured prefix)
//     - This ensures SvelteKit's router uses the correct base path
//
// Without these transformations, resources would fail to load when the app
// is served from a subdirectory (e.g., https://example.com/wolweb/).
func (s *Server) injectBaseHref(htmlContent string) string {
	baseHref := "/"
	if s.Config.URLPrefix != "" {
		baseHref = s.Config.URLPrefix
		if !strings.HasSuffix(baseHref, "/") {
			baseHref += "/"
		}
	}

	// Inject <base> tag as FIRST element in head (before meta charset)
	baseTag := fmt.Sprintf(`<base href="%s">`, baseHref)
	htmlContent = strings.Replace(htmlContent, "<head>\n\t\t<meta", "<head>\n\t\t"+baseTag+"\n\t\t<meta", 1)

	// Convert absolute paths to relative so <base> tag works
	// Replace href="/_app/ with href="_app/ (remove leading slash)
	htmlContent = strings.ReplaceAll(htmlContent, `href="/_app/`, `href="_app/`)
	// Replace href="/favicon with href="favicon (remove leading slash)
	htmlContent = strings.ReplaceAll(htmlContent, `href="/favicon`, `href="favicon`)
	// Replace src="/_app/ with src="_app/
	htmlContent = strings.ReplaceAll(htmlContent, `src="/_app/`, `src="_app/`)

	// Fix JavaScript imports - replace import("/_app/ with import("./_app/
	htmlContent = strings.ReplaceAll(htmlContent, `import("/_app/`, `import("./_app/`)

	// Fix SvelteKit base - replace base: "" with base: "/wolweb" (or "/" if no prefix)
	if s.Config.URLPrefix != "" {
		htmlContent = strings.ReplaceAll(htmlContent, `base: ""`, fmt.Sprintf(`base: "%s"`, s.Config.URLPrefix))
	}

	return htmlContent
}

// serveSPAPage serves an HTML page with appropriate headers and injects base href
func (s *Server) serveSPAPage(w http.ResponseWriter, r *http.Request, filePath string) {
	// Security headers for HTML pages
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	// Strengthened CSP - removed 'unsafe-eval', more restrictive
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; "+
			"script-src 'self' 'unsafe-inline'; "+
			"style-src 'self' 'unsafe-inline'; "+
			"img-src 'self' data: https:; "+
			"connect-src 'self'; "+
			"font-src 'self'; "+
			"frame-ancestors 'none';")

	// No caching for HTML
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Read the HTML file
	htmlBytes, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "Failed to read HTML file", http.StatusInternalServerError)
		return
	}

	// Inject base href and write response
	htmlContent := s.injectBaseHref(string(htmlBytes))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(htmlContent))
}

// serve404Page serves a custom 404 error page using the SPA's error page
func (s *Server) serve404Page(w http.ResponseWriter, r *http.Request, staticPath string) {
	// Security headers for HTML pages
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; "+
			"script-src 'self' 'unsafe-inline'; "+
			"style-src 'self' 'unsafe-inline'; "+
			"img-src 'self' data: https:; "+
			"connect-src 'self'; "+
			"font-src 'self'; "+
			"frame-ancestors 'none';")

	// No caching for error pages
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Try to serve the dedicated 404.html page first
	filePath := filepath.Join(staticPath, "404.html")
	htmlBytes, err := os.ReadFile(filePath)
	if err != nil {
		// Fallback to index.html if 404.html doesn't exist
		filePath = filepath.Join(staticPath, "index.html")
		htmlBytes, err = os.ReadFile(filePath)
		if err != nil {
			// Final fallback to plain text 404
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 Page Not Found"))
			return
		}
	}

	// Inject base href and write response
	htmlContent := s.injectBaseHref(string(htmlBytes))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(htmlContent))
}

// handleConfig returns configuration and authentication status
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	// Config endpoint requires auth only when auth is enabled
	// This allows the frontend to determine auth requirements before login
	if s.Config.UseAuth {
		// When auth is enabled, we still allow unauthenticated access to get basic config
		user := s.getCurrentUser(r)
		response := map[string]interface{}{}

		// Determine if user has readonly access
		isReadOnly := s.Config.ReadOnlyMode || (user != nil && user.ReadOnly)

		// Indicate if interface selection is supported (but don't return interfaces here)
		// Use the dedicated /api/network-interfaces endpoint for that
		if !isReadOnly && s.Config.EnablePerHostInterfaces {
			response["supports_interface_selection"] = true
		}

		// Include user info if authenticated
		if user != nil {
			response["user"] = map[string]interface{}{
				"id":           user.ID,
				"name":         user.Name,
				"readonly":     user.ReadOnly,
				"is_superuser": user.IsSuperuser,
			}
		}

		// Include auth and readonly mode status
		response["use_auth"] = s.Config.UseAuth
		response["readonly_mode"] = s.Config.ReadOnlyMode
		response["os"] = runtime.GOOS
		response["url_prefix"] = s.Config.URLPrefix

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	} else {
		// No auth mode - return basic config
		response := map[string]interface{}{
			"use_auth":      false,
			"readonly_mode": s.Config.ReadOnlyMode,
			"os":            runtime.GOOS,
			"url_prefix":    s.Config.URLPrefix,
		}

		// Indicate if interface selection is supported (but don't return interfaces here)
		// Use the dedicated /api/network-interfaces endpoint for that
		if !s.Config.ReadOnlyMode {
			response["supports_interface_selection"] = true
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// handleNetworkInterfaces returns available network interfaces
func (s *Server) handleNetworkInterfaces(w http.ResponseWriter, r *http.Request) {
	// Network interfaces endpoint requires auth when auth is enabled
	if s.Config.UseAuth && !s.checkAuth(w, r) {
		return
	}

	// Check if use_host_network_interfaces is enabled
	if !s.Config.EnablePerHostInterfaces {
		sendJSONError(w, "Per-host network interface selection is disabled", http.StatusForbidden)
		return
	}

	// Check if in readonly mode
	if s.Config.ReadOnlyMode {
		sendJSONError(w, "Network interface selection not available in readonly mode", http.StatusForbidden)
		return
	}

	// Check if user is readonly (when auth is enabled)
	if s.Config.UseAuth {
		user := s.getCurrentUser(r)
		if user != nil && user.ReadOnly {
			sendJSONError(w, "Network interface selection not available for readonly users", http.StatusForbidden)
			return
		}
	}

	// Get available network interfaces
	interfaces, err := getAvailableInterfaces()
	if err != nil {
		sendJSONError(w, "Failed to get network interfaces", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(interfaces)
}
