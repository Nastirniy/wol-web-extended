package main

import "time"

// Rate limiting constants
const (
	// PingRateLimitPerMinute is the maximum number of ping requests per minute per user
	PingRateLimitPerMinute = 10

	// WoLRateLimitPerMinute is the maximum number of Wake-on-LAN requests per minute per user
	WoLRateLimitPerMinute = 5

	// RateLimiterMaxKeys is the maximum number of keys to track in rate limiter to prevent unbounded memory growth
	RateLimiterMaxKeys = 10000

	// RateLimiterCleanupInterval is how often to clean up old rate limiter entries
	RateLimiterCleanupInterval = 5 * time.Minute

	// RateLimiterCleanupThreshold is how old an entry must be before it's cleaned up
	RateLimiterCleanupThreshold = 1 * time.Hour
)

// WoL history constants
const (
	// MaxWoLHistoryEntries is the maximum number of WoL operations to track for prioritization
	MaxWoLHistoryEntries = 100
)

// Ping timeout constants
const (
	// DefaultPingTimeoutSeconds is the default timeout for ping operations in seconds
	DefaultPingTimeoutSeconds = 2

	// ARPPingTimeoutSeconds is the timeout for ARP ping operations in seconds
	ARPPingTimeoutSeconds = 2

	// PingCacheTTLMultiplier is multiplied by PingTimeout to determine cache TTL
	// Cache TTL = PingTimeout * PingCacheTTLMultiplier seconds
	// This ensures cached results are fresh relative to network conditions
	PingCacheTTLMultiplier = 2
)

// Bulk ping constants
const (
	// BulkPingMultiplier is the multiplier for dynamic rate limiting in bulk ping operations
	// The limit is calculated as: hostCount * BulkPingMultiplier
	BulkPingMultiplier = 2

	// MinBulkPingLimit is the minimum rate limit for bulk ping operations
	MinBulkPingLimit = 10
)

// Session cleanup constants
const (
	// SessionCleanupInterval is how often to clean up expired sessions
	SessionCleanupInterval = 10 * time.Minute
)

// Skeleton display constants
const (
	// MinSkeletonDisplayTime is the minimum time to display loading skeletons to prevent UI flicker
	MinSkeletonDisplayTime = 150 * time.Millisecond
)
