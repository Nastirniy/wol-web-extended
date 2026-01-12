package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
)

type Config struct {
	ListenAddress           string  `json:"listen_address"`             // Combined address:port (e.g., "0.0.0.0:8090", ":8090", "127.0.0.1:3000")
	URLPrefix               string  `json:"url_prefix"`                 // URL prefix for reverse proxy
	DefaultNetworkInterface string  `json:"default_network_interface"`  // Default network interfaces when per-host disabled (Linux only)
	EnablePerHostInterfaces bool    `json:"enable_per_host_interfaces"` // Allow hosts to specify own interfaces (Linux only)
	PingTimeout             int     `json:"ping_timeout_seconds"`       // Ping timeout in seconds
	AuthExpireHours         float64 `json:"auth_expire_hours"`          // Session expiration in hours (supports fractional hours, e.g., 0.5 = 30 minutes)
	UseAuth                 bool    `json:"use_auth"`                   // Enable authentication
	ReadOnlyMode            bool    `json:"readonly_mode"`              // Disable host modifications
	BehindProxy             bool    `json:"behind_proxy"`               // Running behind HTTPS reverse proxy
	Debug                   bool    `json:"debug"`                      // Enable debug logging (deprecated, use log_level instead)
	HealthCheckEnabled      bool    `json:"health_check_enabled"`       // Enable health check endpoint
	// Logging configuration
	LogLevel      string `json:"log_level"`        // Log level: "debug", "info", "warning", "error" (default: "info")
	LogOutputMode string `json:"log_output_mode"`  // Log output: "stdout", "file", "both" (default: "stdout")
	LogDir        string `json:"log_dir"`          // Directory for log files when output_mode is "file" or "both"
	LogMaxSizeMB  int    `json:"log_max_size_mb"`  // Max log file size in MB before rotation (0 = no limit, default: 100)
	LogMaxAgeDays int    `json:"log_max_age_days"` // Max days to keep old log files (0 = keep all, default: 30)
	LogRotation   bool   `json:"log_rotation"`     // Enable log rotation (default: true)
}


func loadConfig(configPath string) *Config {
	config := &Config{
		ListenAddress:           ":8090", // Default listen on all interfaces, port 8090
		URLPrefix:               "",
		DefaultNetworkInterface: "",
		EnablePerHostInterfaces: false,
		PingTimeout:             5,
		AuthExpireHours:         4,
		UseAuth:                 true,
		ReadOnlyMode:            false,
		BehindProxy:             false,
		Debug:                   false,
		HealthCheckEnabled:      true,
		// Logging defaults - stdout for development, file for production/systemd
		LogLevel:      "info",
		LogOutputMode: "stdout", // Can be: "stdout", "file", or "both"
		LogDir:        "./logs",
		LogMaxSizeMB:  100,
		LogMaxAgeDays: 30,
		LogRotation:   true,
	}

	// Use provided config path or default to config.json
	if configPath == "" {
		configPath = "./config.json"
	}

	// Load from config file if it exists (but skip sensitive fields)
	if configFile, err := os.ReadFile(configPath); err == nil {
		tempConfig := &Config{}
		if err := json.Unmarshal(configFile, tempConfig); err != nil {
			Fatal("Failed to parse config file %s: %v", configPath, err)
		}
		config.ListenAddress = tempConfig.ListenAddress
		config.URLPrefix = tempConfig.URLPrefix
		config.DefaultNetworkInterface = tempConfig.DefaultNetworkInterface
		config.EnablePerHostInterfaces = tempConfig.EnablePerHostInterfaces
		config.PingTimeout = tempConfig.PingTimeout
		if tempConfig.AuthExpireHours > 0 {
			config.AuthExpireHours = tempConfig.AuthExpireHours
		}
		config.UseAuth = tempConfig.UseAuth
		config.ReadOnlyMode = tempConfig.ReadOnlyMode
		config.BehindProxy = tempConfig.BehindProxy
		config.Debug = tempConfig.Debug
		config.HealthCheckEnabled = tempConfig.HealthCheckEnabled
		// Load logging configuration
		if tempConfig.LogLevel != "" {
			config.LogLevel = tempConfig.LogLevel
		}
		if tempConfig.LogOutputMode != "" {
			config.LogOutputMode = tempConfig.LogOutputMode
		}
		if tempConfig.LogDir != "" {
			config.LogDir = tempConfig.LogDir
		}
		if tempConfig.LogMaxSizeMB > 0 {
			config.LogMaxSizeMB = tempConfig.LogMaxSizeMB
		}
		if tempConfig.LogMaxAgeDays >= 0 {
			config.LogMaxAgeDays = tempConfig.LogMaxAgeDays
		}
		config.LogRotation = tempConfig.LogRotation
		Info("Loaded configuration from: %s", configPath)
	} else if !os.IsNotExist(err) {
		Fatal("Failed to read config file %s: %v", configPath, err)
	} else {
		Info("Config file not found at: %s, using defaults", configPath)
	}

	// Environment variable overrides
	if listenAddr := os.Getenv("LISTEN_ADDRESS"); listenAddr != "" {
		config.ListenAddress = listenAddr
	}

	if urlPrefix := os.Getenv("URL_PREFIX"); urlPrefix != "" {
		config.URLPrefix = urlPrefix
	}

	if defaultInterface := os.Getenv("DEFAULT_NETWORK_INTERFACE"); defaultInterface != "" {
		config.DefaultNetworkInterface = defaultInterface
	}

	if enablePerHost := os.Getenv("ENABLE_PER_HOST_INTERFACES"); enablePerHost != "" {
		config.EnablePerHostInterfaces = enablePerHost == "true" || enablePerHost == "1"
	}

	if pingTimeout := os.Getenv("PING_TIMEOUT_SECONDS"); pingTimeout != "" {
		if timeout, err := strconv.Atoi(pingTimeout); err == nil {
			config.PingTimeout = timeout
		} else {
			Warning("Invalid PING_TIMEOUT_SECONDS value '%s', using default: %d", pingTimeout, config.PingTimeout)
		}
	}

	if authExpire := os.Getenv("AUTH_EXPIRE_HOURS"); authExpire != "" {
		if hours, err := strconv.ParseFloat(authExpire, 64); err == nil {
			config.AuthExpireHours = hours
		} else {
			Warning("Invalid AUTH_EXPIRE_HOURS value '%s', using default: %.2f", authExpire, config.AuthExpireHours)
		}
	}

	if useAuth := os.Getenv("USE_AUTH"); useAuth != "" {
		config.UseAuth = useAuth == "true" || useAuth == "1"
	}

	if readOnly := os.Getenv("READONLY_MODE"); readOnly != "" {
		config.ReadOnlyMode = readOnly == "true" || readOnly == "1"
	}

	if behindProxy := os.Getenv("BEHIND_PROXY"); behindProxy != "" {
		config.BehindProxy = behindProxy == "true" || behindProxy == "1"
	}

	if debug := os.Getenv("DEBUG"); debug != "" {
		config.Debug = debug == "true" || debug == "1"
	}

	if healthCheck := os.Getenv("HEALTH_CHECK_ENABLED"); healthCheck != "" {
		config.HealthCheckEnabled = healthCheck == "true" || healthCheck == "1"
	}

	// Logging environment variables
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.LogLevel = logLevel
	}

	if logOutputMode := os.Getenv("LOG_OUTPUT_MODE"); logOutputMode != "" {
		config.LogOutputMode = logOutputMode
	}

	if logDir := os.Getenv("LOG_DIR"); logDir != "" {
		config.LogDir = logDir
	}

	if logMaxSize := os.Getenv("LOG_MAX_SIZE_MB"); logMaxSize != "" {
		if size, err := strconv.Atoi(logMaxSize); err == nil {
			config.LogMaxSizeMB = size
		} else {
			Warning("Invalid LOG_MAX_SIZE_MB value '%s', using default: %d", logMaxSize, config.LogMaxSizeMB)
		}
	}

	if logMaxAge := os.Getenv("LOG_MAX_AGE_DAYS"); logMaxAge != "" {
		if age, err := strconv.Atoi(logMaxAge); err == nil {
			config.LogMaxAgeDays = age
		} else {
			Warning("Invalid LOG_MAX_AGE_DAYS value '%s', using default: %d", logMaxAge, config.LogMaxAgeDays)
		}
	}

	if logRotation := os.Getenv("LOG_ROTATION"); logRotation != "" {
		config.LogRotation = logRotation == "true" || logRotation == "1"
	}

	// Handle legacy Debug flag - if Debug is true, set LogLevel to debug
	if config.Debug {
		config.LogLevel = "debug"
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		Fatal("Invalid configuration: %v", err)
	}

	return config
}

// validateConfig validates the configuration values
func validateConfig(c *Config) error {
	// Validate listen_address format
	if c.ListenAddress == "" {
		return fmt.Errorf("listen_address cannot be empty")
	}

	// Parse and validate the address format
	host, port, err := net.SplitHostPort(c.ListenAddress)
	if err != nil {
		return fmt.Errorf("invalid listen_address format '%s': %v (expected format: 'addr:port' or ':port')", c.ListenAddress, err)
	}

	// Validate port
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return fmt.Errorf("invalid port in listen_address '%s': must be 1-65535", c.ListenAddress)
	}

	// Validate host if provided (empty is valid, means all interfaces)
	if host != "" {
		// Try to parse as IP
		if ip := net.ParseIP(host); ip == nil {
			// If not a valid IP, check if it's a valid hostname/interface
			// We allow it to pass through - net.Listen will catch invalid addresses
			Warning("listen_address host '%s' is not a valid IP address. Will attempt to bind anyway.", host)
		}
	}

	if c.AuthExpireHours <= 0 {
		return fmt.Errorf("auth_expire_hours must be greater than 0, got: %.2f", c.AuthExpireHours)
	}

	if c.PingTimeout < 1 || c.PingTimeout > 60 {
		return fmt.Errorf("ping_timeout_seconds must be between 1-60, got: %d", c.PingTimeout)
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug":   true,
		"info":    true,
		"warning": true,
		"error":   true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log_level '%s': must be one of: debug, info, warning, error", c.LogLevel)
	}

	// Validate log output mode
	validOutputModes := map[string]bool{
		"stdout": true,
		"file":   true,
		"both":   true,
	}
	if !validOutputModes[c.LogOutputMode] {
		return fmt.Errorf("invalid log_output_mode '%s': must be one of: stdout, file, both", c.LogOutputMode)
	}

	// Validate log directory when needed
	if (c.LogOutputMode == "file" || c.LogOutputMode == "both") && c.LogDir == "" {
		return fmt.Errorf("log_dir must be specified when log_output_mode is '%s'", c.LogOutputMode)
	}

	// Validate log rotation settings
	if c.LogMaxSizeMB < 0 {
		return fmt.Errorf("log_max_size_mb must be >= 0, got: %d", c.LogMaxSizeMB)
	}

	if c.LogMaxAgeDays < 0 {
		return fmt.Errorf("log_max_age_days must be >= 0, got: %d", c.LogMaxAgeDays)
	}

	return nil
}

func getAvailableInterfaces() ([]NetworkInterfaceInfo, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var result []NetworkInterfaceInfo
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue // Skip down or loopback interfaces
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ipNet.IP.To4() != nil { // IPv4
					result = append(result, NetworkInterfaceInfo{
						Name: iface.Name,
						IP:   ipNet.IP.String(),
					})
				}
			}
		}
	}

	return result, nil
}

type NetworkInterfaceInfo struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

func createDefaultConfig(configPath string) error {
	config := &Config{
		ListenAddress:           ":8090",
		URLPrefix:               "",
		DefaultNetworkInterface: "",
		EnablePerHostInterfaces: false,
		PingTimeout:             5,
		AuthExpireHours:         4,
		UseAuth:                 true,
		ReadOnlyMode:            false,
		BehindProxy:             false,
		Debug:                   false,
		HealthCheckEnabled:      true,
		// Logging configuration
		LogLevel:      "info",
		LogOutputMode: "stdout",
		LogDir:        "./logs",
		LogMaxSizeMB:  100,
		LogMaxAgeDays: 30,
		LogRotation:   true,
	}

	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", configPath, err)
	}

	Info("Created default config file: %s", configPath)
	return nil
}
