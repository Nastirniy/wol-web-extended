package main

import (
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Password  string    `json:"-"`
	ReadOnly  bool      `json:"readonly"`
	IsSuperuser bool    `json:"is_superuser"`
	Created   time.Time `json:"created"`
	Updated   time.Time `json:"updated"`
}

type Host struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	MAC             string     `json:"mac"`
	Broadcast       string     `json:"broadcast"`
	Interface       string     `json:"interface"`
	StaticIP        string     `json:"static_ip"`        // Static IPv4 address - manually specified IP for ping/WoL (ignores ARP when not using fallback)
	UseAsFallback   bool       `json:"use_as_fallback"`  // If true, use static IP only when ARP resolution fails or host not responding
	UserID          *string    `json:"user"`
	Created         time.Time  `json:"created"`
	Updated         time.Time  `json:"updated"`
}

type Server struct {
	DB            *sql.DB
	Config        *Config
	PingRateLimit *RateLimiter
	WoLRateLimit  *RateLimiter
	WoLHistory    *WoLHistory
	PingCache     *PingCache
}

type WoLHistory struct {
	mutex       sync.RWMutex
	recentWoL   map[string]time.Time // hostID -> last WoL time
	maxEntries  int
}

func initDatabase(dbPath string) (*sql.DB, error) {
	// Add query parameters for SQLite configuration
	// - _journal_mode=WAL: Write-Ahead Logging for better concurrency
	// - _busy_timeout=5000: Wait up to 5 seconds if database is locked
	// - _synchronous=NORMAL: Balance between safety and performance
	// - _cache_size=1000: Cache size in pages
	// - _foreign_keys=1: Enable foreign key constraints
	dbPath = dbPath + "?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_cache_size=1000&_foreign_keys=1"

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Configure connection pool for better resource management
	// Limit to 1 connection for SQLite to avoid locking issues
	db.SetMaxOpenConns(1) // SQLite works best with single writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)                // No max lifetime
	db.SetConnMaxIdleTime(30 * time.Second) // Maximum idle connection time

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Initialize database schema
	if err := initSchema(db); err != nil {
		return nil, fmt.Errorf("database schema initialization failed: %w", err)
	}

	Info("Database initialized with WAL mode and optimized for concurrent access")
	return db, nil
}


// NewWoLHistory creates a new WoL history tracker
func NewWoLHistory(maxEntries int) *WoLHistory {
	return &WoLHistory{
		recentWoL:  make(map[string]time.Time),
		maxEntries: maxEntries,
	}
}

// RecordWoL records when a host was sent WoL packet
func (wh *WoLHistory) RecordWoL(hostID string) {
	wh.mutex.Lock()
	defer wh.mutex.Unlock()

	wh.recentWoL[hostID] = time.Now()

	// Clean up old entries if we exceed max entries
	if len(wh.recentWoL) > wh.maxEntries {
		// Find oldest entry to remove
		oldestTime := time.Now()
		oldestHost := ""
		for host, t := range wh.recentWoL {
			if t.Before(oldestTime) {
				oldestTime = t
				oldestHost = host
			}
		}
		if oldestHost != "" {
			delete(wh.recentWoL, oldestHost)
		}
	}
}

// GetWoLTime returns when a host was last sent WoL packet
func (wh *WoLHistory) GetWoLTime(hostID string) (time.Time, bool) {
	wh.mutex.RLock()
	defer wh.mutex.RUnlock()

	t, exists := wh.recentWoL[hostID]
	return t, exists
}

// SortHostsByWoLPriority sorts hosts putting recent WoL hosts first.
//
// Sorting priority:
//  1. Hosts with recent WoL packets (most recent first)
//  2. Hosts without WoL history (sorted by creation date, newest first)
//
// Rationale: Recently woken hosts are more likely to be online and actively used,
// so they should be pinged first in bulk ping operations for better perceived performance.
//
// Returns a new sorted slice without modifying the original.
func (wh *WoLHistory) SortHostsByWoLPriority(hosts []Host) []Host {
	wh.mutex.RLock()
	defer wh.mutex.RUnlock()

	// Create a copy to avoid modifying original slice
	sorted := make([]Host, len(hosts))
	copy(sorted, hosts)

	// Sort by WoL priority (recent WoL first, then by creation date)
	sort.Slice(sorted, func(i, j int) bool {
		timeI, hasI := wh.recentWoL[sorted[i].ID]
		timeJ, hasJ := wh.recentWoL[sorted[j].ID]

		// If both have WoL history, sort by most recent
		if hasI && hasJ {
			return timeI.After(timeJ)
		}

		// WoL hosts come first
		if hasI && !hasJ {
			return true
		}
		if !hasI && hasJ {
			return false
		}

		// If neither has WoL history, sort by creation date
		return sorted[i].Created.After(sorted[j].Created)
	})

	return sorted
}

// seedDatabase is intentionally empty for security reasons.
//
// Security rationale:
//   - No default credentials prevents unauthorized access from default/well-known usernames
//   - Forces explicit superuser creation via secure setup flow
//   - Prevents accidental production deployment with hardcoded test credentials
//
// Superuser creation methods:
//   1. Web UI: Visit /auth when no superuser exists to show setup form
//   2. API: POST to /api/auth/setup with {"username": "...", "password": "..."}
//
// Password reset:
//   - CLI: Use --reset-admin flag to interactively reset a superuser's password
//
// This empty function is called during table creation (createTables) and is kept
// to maintain a clear extension point if seed data is needed in the future.
func seedDatabase(db *sql.DB) error {
	return nil
}

func hashPassword(password string) string {
	// Use bcrypt with default cost (currently 10)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		Error("Failed to hash password: %v", err)
		// Fallback to a very high cost to prevent timing attacks
		hash, _ = bcrypt.GenerateFromPassword([]byte(password), bcrypt.MaxCost)
	}
	return string(hash)
}

func verifyPassword(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}