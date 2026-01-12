package main

import (
	"sync"
	"time"
)

// PingCacheEntry stores cached ping results with expiration
type PingCacheEntry struct {
	PingSuccess bool
	ARPSuccess  bool
	Timestamp   time.Time
	InProgress  bool            // Indicates if a ping is currently in progress
	WaitChan    chan pingResult // Channel for request coalescing
}

// PingCache manages cached ping results to reduce redundant network operations
type PingCache struct {
	cache      map[string]*PingCacheEntry // Key: host_id or MAC address
	cacheMutex sync.RWMutex
	ttl        time.Duration // How long to keep cached results
}

type pingResult struct {
	PingSuccess bool
	ARPSuccess  bool
	Error       error
}

// NewPingCache creates a new ping cache with specified TTL
func NewPingCache(ttl time.Duration) *PingCache {
	pc := &PingCache{
		cache: make(map[string]*PingCacheEntry),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go pc.cleanupExpiredEntries()

	return pc
}

// Get retrieves cached ping result if valid, returns nil if cache miss or expired
func (pc *PingCache) Get(key string) *PingCacheEntry {
	pc.cacheMutex.RLock()
	defer pc.cacheMutex.RUnlock()

	entry, exists := pc.cache[key]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Since(entry.Timestamp) > pc.ttl {
		return nil
	}

	return entry
}

// StartPing marks a ping operation as in-progress and returns a wait channel
// This enables request coalescing - multiple concurrent requests for same host
// will wait for the same ping operation to complete
func (pc *PingCache) StartPing(key string) (isFirstRequest bool, waitChan chan pingResult) {
	pc.cacheMutex.Lock()
	defer pc.cacheMutex.Unlock()

	entry, exists := pc.cache[key]

	// Check if there's already a ping in progress
	if exists && entry.InProgress {
		// Return existing wait channel - this request will coalesce with ongoing ping
		return false, entry.WaitChan
	}

	// Start new ping operation
	waitChan = make(chan pingResult, 10) // Buffered to avoid blocking
	pc.cache[key] = &PingCacheEntry{
		InProgress: true,
		WaitChan:   waitChan,
		Timestamp:  time.Now(),
	}

	return true, waitChan
}

// Set stores a ping result in the cache
func (pc *PingCache) Set(key string, pingSuccess, arpSuccess bool) {
	pc.cacheMutex.Lock()
	defer pc.cacheMutex.Unlock()

	entry, exists := pc.cache[key]

	// If there was a ping in progress, notify all waiters
	if exists && entry.InProgress {
		result := pingResult{
			PingSuccess: pingSuccess,
			ARPSuccess:  arpSuccess,
			Error:       nil,
		}

		// Notify all waiting requests (non-blocking)
		select {
		case entry.WaitChan <- result:
		default:
			// Channel full or no receivers, continue
		}
		close(entry.WaitChan)
	}

	// Store the result
	pc.cache[key] = &PingCacheEntry{
		PingSuccess: pingSuccess,
		ARPSuccess:  arpSuccess,
		Timestamp:   time.Now(),
		InProgress:  false,
		WaitChan:    nil,
	}
}

// SetError marks a ping operation as failed
func (pc *PingCache) SetError(key string, err error) {
	pc.cacheMutex.Lock()
	defer pc.cacheMutex.Unlock()

	entry, exists := pc.cache[key]

	// If there was a ping in progress, notify all waiters
	if exists && entry.InProgress {
		result := pingResult{
			PingSuccess: false,
			ARPSuccess:  false,
			Error:       err,
		}

		// Notify all waiting requests (non-blocking)
		select {
		case entry.WaitChan <- result:
		default:
		}
		close(entry.WaitChan)
	}

	// Remove from cache on error (don't cache failures long-term)
	delete(pc.cache, key)
}

// Invalidate removes a cache entry (useful after sending WoL packet)
func (pc *PingCache) Invalidate(key string) {
	pc.cacheMutex.Lock()
	defer pc.cacheMutex.Unlock()

	delete(pc.cache, key)
}

// InvalidateAll clears all cache entries
func (pc *PingCache) InvalidateAll() {
	pc.cacheMutex.Lock()
	defer pc.cacheMutex.Unlock()

	pc.cache = make(map[string]*PingCacheEntry)
}

// cleanupExpiredEntries periodically removes expired cache entries
func (pc *PingCache) cleanupExpiredEntries() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		pc.cacheMutex.Lock()
		now := time.Now()

		for key, entry := range pc.cache {
			// Remove expired entries (but keep in-progress pings)
			if !entry.InProgress && now.Sub(entry.Timestamp) > pc.ttl {
				delete(pc.cache, key)
			}
		}

		pc.cacheMutex.Unlock()
	}
}

// GetStats returns cache statistics for monitoring
func (pc *PingCache) GetStats() map[string]interface{} {
	pc.cacheMutex.RLock()
	defer pc.cacheMutex.RUnlock()

	inProgressCount := 0
	cachedCount := 0

	for _, entry := range pc.cache {
		if entry.InProgress {
			inProgressCount++
		} else {
			cachedCount++
		}
	}

	return map[string]interface{}{
		"total_entries":    len(pc.cache),
		"in_progress":      inProgressCount,
		"cached_results":   cachedCount,
		"ttl_seconds":      pc.ttl.Seconds(),
	}
}
