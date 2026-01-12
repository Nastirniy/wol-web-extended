package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
)

// RateLimiter implements a rate limiter with memory leak prevention
type RateLimiter struct {
	requests map[string]*requestWindow
	mutex    sync.RWMutex
	limit    int
	window   time.Duration
	maxKeys  int
}

type requestWindow struct {
	times      []time.Time
	lastAccess time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string]*requestWindow),
		limit:    limit,
		window:   window,
		maxKeys:  RateLimiterMaxKeys,
	}
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Limit total number of tracked keys
	if len(rl.requests) > rl.maxKeys {
		rl.cleanupOldest(now)
	}

	// Get existing request window for this key
	window, exists := rl.requests[key]
	if !exists {
		window = &requestWindow{
			times:      make([]time.Time, 0),
			lastAccess: now,
		}
		rl.requests[key] = window
	}

	// Update last access time
	window.lastAccess = now

	// Remove requests outside the time window
	var validRequests []time.Time
	for _, req := range window.times {
		if now.Sub(req) < rl.window {
			validRequests = append(validRequests, req)
		}
	}

	// Check if we're within the limit
	if len(validRequests) >= rl.limit {
		window.times = validRequests
		return false
	}

	// Add this request
	validRequests = append(validRequests, now)
	window.times = validRequests

	return true
}

// cleanupOldest removes entries not accessed within the cleanup threshold
func (rl *RateLimiter) cleanupOldest(now time.Time) {
	threshold := now.Add(-1 * RateLimiterCleanupThreshold)
	for key, window := range rl.requests {
		if window.lastAccess.Before(threshold) {
			delete(rl.requests, key)
		}
	}
}

// CleanupOldEntries removes old entries to prevent memory leaks
func (rl *RateLimiter) CleanupOldEntries() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	for key, window := range rl.requests {
		var validRequests []time.Time
		for _, req := range window.times {
			if now.Sub(req) < rl.window {
				validRequests = append(validRequests, req)
			}
		}

		if len(validRequests) == 0 {
			delete(rl.requests, key)
		} else {
			window.times = validRequests
		}
	}
}

func printHelp() {
	fmt.Println("Wake-on-LAN Web Server")
	fmt.Println("======================")
	fmt.Println()
	fmt.Println("DESCRIPTION:")
	fmt.Println("  A web-based Wake-on-LAN management system with user authentication,")
	fmt.Println("  device status monitoring, and per-host network interface selection.")
	fmt.Println()
	fmt.Println("PLATFORM SUPPORT:")
	fmt.Println("  PRIMARY: Linux (full functionality)")
	fmt.Println("  LIMITED: Windows, macOS (basic WoL only)")
	fmt.Println()
	fmt.Println("  Linux-only features:")
	fmt.Println("    - ARP discovery and active scanning")
	fmt.Println("    - Per-host network interface selection")
	fmt.Println("    - MAC address detection and validation")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Printf("  %s [OPTIONS]\n", os.Args[0])
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  -h, --help              Show this help message and exit")
	fmt.Println("  -config <path>          Path to config.json file (default: ./config.json)")
	fmt.Println("  -db <path>              Path to database file (default: ./wol.db)")
	fmt.Println("  -debug                  Enable debug logging (overrides config)")
	fmt.Println("  --reset-admin           Reset password for a superuser (interactive)")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Run with defaults")
	fmt.Printf("  %s\n", os.Args[0])
	fmt.Println()
	fmt.Println("  # Custom paths with debug logging")
	fmt.Printf("  %s -config /etc/wol/config.json -db /var/lib/wol/data.db -debug\n", os.Args[0])
	fmt.Println()
	fmt.Println("  # Reset superuser password (interactive)")
	fmt.Printf("  %s --reset-admin\n", os.Args[0])
	fmt.Println()
	fmt.Println("  # Run as systemd service")
	fmt.Println("  sudo systemctl start wolweb")
	fmt.Println()
	fmt.Println("CONFIGURATION:")
	fmt.Println("  Config file (config.json) fields:")
	fmt.Println("    listen_address               Bind address (e.g., ':8090', '0.0.0.0:8090', '127.0.0.1:3000')")
	fmt.Println("    url_prefix                   URL prefix for reverse proxy (e.g., '/wolweb')")
	fmt.Println("    default_network_interface    Default network interface when per-host disabled")
	fmt.Println("    enable_per_host_interfaces   Allow hosts to specify own interfaces (true/false)")
	fmt.Println("    ping_timeout_seconds         Ping timeout in seconds (1-60)")
	fmt.Println("    auth_expire_hours            Session expiration in hours")
	fmt.Println("    use_auth                     Enable authentication (true/false)")
	fmt.Println("    readonly_mode                Disable host modifications (true/false)")
	fmt.Println("    behind_proxy                 Running behind HTTPS proxy (true/false)")
	fmt.Println("    debug                        Enable debug logging (true/false)")
	fmt.Println()
	fmt.Println("  Environment variables (override config file):")
	fmt.Println("    LISTEN_ADDRESS               Server listen address (e.g., ':8090')")
	fmt.Println("    URL_PREFIX                   URL prefix for the application")
	fmt.Println("    DEFAULT_NETWORK_INTERFACE    Default network interface(s)")
	fmt.Println("    ENABLE_PER_HOST_INTERFACES   Allow per-host interfaces (true/1)")
	fmt.Println("    PING_TIMEOUT_SECONDS         Ping timeout in seconds")
	fmt.Println("    AUTH_EXPIRE_HOURS            Session expiration in hours")
	fmt.Println("    USE_AUTH                     Enable authentication (true/1)")
	fmt.Println("    READONLY_MODE                Disable host modifications (true/1)")
	fmt.Println("    BEHIND_PROXY                 Behind reverse proxy (true/1)")
	fmt.Println("    DEBUG                        Enable debug logging (true/1)")
	fmt.Println()
	fmt.Println("NETWORK INTERFACE MODES:")
	fmt.Println("  1. Global Interface (enable_per_host_interfaces: false, default)")
	fmt.Println("     - All hosts use default_network_interface setting")
	fmt.Println("     - If default_network_interface empty, uses all available interfaces")
	fmt.Println("     - Simplest configuration")
	fmt.Println()
	fmt.Println("  2. Per-Host Selection (enable_per_host_interfaces: true)")
	fmt.Println("     - Each host can specify its own interface(s)")
	fmt.Println("     - If host has no interface, uses all available interfaces")
	fmt.Println("     - Works with both auth and no-auth modes")
	fmt.Println("     - Disabled in readonly_mode")
	fmt.Println()
	fmt.Println("FIRST TIME SETUP:")
	fmt.Println("  1. Start the server")
	fmt.Println("  2. Navigate to http://localhost:8090/auth")
	fmt.Println("  3. Create first superuser via web UI or API:")
	fmt.Println("     curl -X POST http://localhost:8090/api/auth/setup \\")
	fmt.Println("       -H 'Content-Type: application/json' \\")
	fmt.Println("       -d '{\"username\":\"admin\",\"password\":\"yourpassword\"}'")
	fmt.Println()
	fmt.Println("TROUBLESHOOTING:")
	fmt.Println("  - Enable debug mode: -debug flag or DEBUG=true env var")
	fmt.Println("  - Check permissions: Ensure read/write access to config and database")
	fmt.Println("  - WoL not working: Verify network interface and broadcast address")
	fmt.Println("  - ARP discovery fails: Ensure CAP_NET_RAW capability on Linux")
	fmt.Println("    sudo setcap cap_net_raw+ep /path/to/wolweb")
	fmt.Println()
	fmt.Println("MORE INFO:")
	fmt.Println("  GitHub: https://github.com/Nastirniy/wol-web-extended")
	fmt.Println("  Docs:   See CLAUDE.md and README.md in repository")
}


func checkFilePermissions(path string, needWrite bool) error {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - check if we can create it
			if needWrite {
				// Try to create and remove a test file
				testFile, err := os.Create(path)
				if err != nil {
					return fmt.Errorf("cannot create file: %w", err)
				}
				testFile.Close()
				os.Remove(path)
				return nil
			}
			return fmt.Errorf("file does not exist and cannot be created")
		}
		return fmt.Errorf("cannot access file: %w", err)
	}

	// Check if it's a directory
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}

	// Check read permissions
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}
	file.Close()

	// Check write permissions if needed
	if needWrite {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
		if err != nil {
			return fmt.Errorf("cannot write to file: %w", err)
		}
		file.Close()
	}

	return nil
}

func main() {
	// Initialize rate limiters
	pingRateLimiter := NewRateLimiter(PingRateLimitPerMinute, time.Minute)
	wolRateLimiter := NewRateLimiter(WoLRateLimitPerMinute, time.Minute)

	// Start cleanup goroutine for rate limiters
	go func() {
		ticker := time.NewTicker(RateLimiterCleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			pingRateLimiter.CleanupOldEntries()
			wolRateLimiter.CleanupOldEntries()
		}
	}()

	// Parse command line arguments
	configPath := "./config.json"
	dbPath := "./wol.db"
	resetAdmin := false
	showHelp := false
	debugFlag := false

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			showHelp = true
		case "-config":
			if i+1 < len(args) {
				configPath = args[i+1]
				i++
			} else {
				Fatal("Error: -config requires a path argument")
			}
		case "-db":
			if i+1 < len(args) {
				dbPath = args[i+1]
				i++
			} else {
				Fatal("Error: -db requires a path argument")
			}
		case "-debug", "--debug":
			debugFlag = true
		case "--reset-admin":
			resetAdmin = true
		default:
			if args[i] != "" && args[i][0] == '-' {
				Warning("Unknown flag '%s' (use -h or --help for usage)", args[i])
			}
		}
	}

	// Show help and exit
	if showHelp {
		printHelp()
		return
	}

	// Check file permissions BEFORE loading anything
	Info("Checking file permissions...")

	// Check config file permissions (read only needed)
	if _, err := os.Stat(configPath); err == nil {
		if err := checkFilePermissions(configPath, false); err != nil {
			Fatal("PERMISSION ERROR - Config file (%s): %v", configPath, err)
		}
		Info("OK - Config file readable: %s", configPath)
	} else {
		// Config doesn't exist - check if we can create it
		if err := checkFilePermissions(configPath, true); err != nil {
			Warning("Cannot create config file (%s): %v", configPath, err)
			Warning("Proceeding with default configuration")
		} else {
			Info("OK - Can create config file: %s", configPath)
		}
	}

	// Check database permissions (read/write needed)
	if err := checkFilePermissions(dbPath, true); err != nil {
		Fatal("PERMISSION ERROR - Database file (%s): %v\nHint: Ensure the process has read/write permissions to this file and directory", dbPath, err)
	}
	Info("OK - Database file accessible: %s", dbPath)

	// Load configuration
	config := loadConfig(configPath)

	// Enable debug mode from CLI flag or config
	if debugFlag {
		config.Debug = true
		config.LogLevel = "debug"
	}

	// Initialize logger based on configuration
	logLevel := LogLevelInfo
	switch config.LogLevel {
	case "debug":
		logLevel = LogLevelDebug
	case "info":
		logLevel = LogLevelInfo
	case "warning":
		logLevel = LogLevelWarning
	case "error":
		logLevel = LogLevelError
	}

	loggerConfig := LoggerConfig{
		Level:           logLevel,
		OutputMode:      config.LogOutputMode,
		LogDir:          config.LogDir,
		MaxFileSizeMB:   config.LogMaxSizeMB,
		MaxAgeDays:      config.LogMaxAgeDays,
		RotationEnabled: config.LogRotation,
	}

	if err := InitLogger(loggerConfig); err != nil {
		Fatal("Failed to initialize logger: %v", err)
	}
	defer GetLogger().Close()

	if config.LogLevel == "debug" {
		Info("DEBUG MODE ENABLED")
	}

	// Create default config file if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := createDefaultConfig(configPath); err != nil {
			Warning("Could not create default config: %v", err)
		} else {
			Debug("Created default config file: %s", configPath)
			config = loadConfig(configPath)
		}
	}

	// Initialize database
	db, err := initDatabase(dbPath)
	if err != nil {
		Fatal("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Handle --reset-admin flag
	if resetAdmin {
		if err := resetAdminPassword(db); err != nil {
			Fatal("Password reset failed: %v", err)
		}
		return
	}

	// Create server with database and configuration
	pingCacheTTL := time.Duration(config.PingTimeout*PingCacheTTLMultiplier) * time.Second
	server := &Server{
		DB:            db,
		Config:        config,
		PingRateLimit: pingRateLimiter,
		WoLRateLimit:  wolRateLimiter,
		WoLHistory:    NewWoLHistory(MaxWoLHistoryEntries),
		PingCache:     NewPingCache(pingCacheTTL),
	}

	// Start session cleanup goroutine if auth is enabled
	if config.UseAuth {
		go func() {
			ticker := time.NewTicker(SessionCleanupInterval)
			defer ticker.Stop()
			for range ticker.C {
				server.cleanupExpiredSessions()
			}
		}()
	}

	// Setup routes
	router := mux.NewRouter()
	server.setupRoutes(router)

	// Start server
	Info("=================================================")
	Info("Starting Wake-on-LAN Web Server v%s", Version)
	Info("=================================================")
	Info("Listen address:    %s", config.ListenAddress)
	Info("Configuration:     %s", configPath)
	Info("Database:          %s", dbPath)
	Info("Authentication:    %v", config.UseAuth)
	Info("Log level:         %s", config.LogLevel)
	Info("Log output:        %s", config.LogOutputMode)
	if config.LogOutputMode != "stdout" {
		Info("Log directory:     %s", config.LogDir)
		Info("Log rotation:      %v (max size: %dMB, max age: %dd)",
			config.LogRotation, config.LogMaxSizeMB, config.LogMaxAgeDays)
	}
	Info("URL prefix:        %s", func() string {
		if config.URLPrefix == "" {
			return "(none)"
		}
		return config.URLPrefix
	}())
	Debug("Network interface: %s", func() string {
		if config.DefaultNetworkInterface == "" {
			return "(all interfaces)"
		}
		return config.DefaultNetworkInterface
	}())
	Debug("Per-host interfaces: %v", config.EnablePerHostInterfaces)

	// Warn if running on non-Linux system
	if runtime.GOOS != "linux" {
		Warning("Running on %s platform", runtime.GOOS)
		Warning("This application is designed for Linux systems")
		Warning("Missing Linux-specific dependencies:")
		Warning("  - Raw socket support (syscall.AF_PACKET)")
		Warning("  - ARP packet handling capabilities")
		Warning("  - Network interface binding for packet-level operations")
		Warning("Core functionality (ARP discovery, per-host network interfaces) will NOT work on %s", runtime.GOOS)
		Warning("Only basic Wake-on-LAN functionality will be available")
	} else {
		// On Linux, check if we have necessary capabilities
		Info("Running on Linux - full functionality available")
		Info("Required Linux capabilities:")
		Info("  - CAP_NET_RAW (for ARP discovery)")
		Info("  - CAP_NET_ADMIN (for ARP cache flushing)")
		Info("  - Raw socket access (AF_PACKET)")
		Info("If ARP discovery fails, ensure the process has CAP_NET_RAW capability")
		Info("Run with: sudo setcap cap_net_raw,cap_net_admin+ep /path/to/wolweb")
	}

	// Security warning for insecure deployment
	if config.UseAuth && !config.BehindProxy {
		Warning("Authentication is enabled but BehindProxy is false")
		Warning("Service is running without HTTPS protection")
		Warning("Credentials and session cookies will be transmitted in PLAINTEXT")
		Warning("This configuration is INSECURE for production use")
		Warning("Set 'behind_proxy: true' in config.json when using reverse proxy with HTTPS")
	}

	Info("=================================================")
	Info("Server ready - Access at: http://%s%s", config.ListenAddress, config.URLPrefix)
	Info("=================================================")

	if err := http.ListenAndServe(config.ListenAddress, router); err != nil {
		Fatal("Failed to start server: %v", err)
	}
}
