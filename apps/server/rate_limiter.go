package main

import (
	"time"
)

// checkDynamicRateLimit checks if a request is allowed under dynamic rate limiting.
// limit: maximum number of requests allowed in the time window
// window: time duration for the rate limit window
func (s *Server) checkDynamicRateLimit(userKey string, limit int, window time.Duration) bool {
	s.PingRateLimit.mutex.Lock()
	defer s.PingRateLimit.mutex.Unlock()

	now := time.Now()

	// Get existing request window for this key
	reqWindow, exists := s.PingRateLimit.requests[userKey]
	if !exists {
		reqWindow = &requestWindow{
			times:      make([]time.Time, 0),
			lastAccess: now,
		}
		s.PingRateLimit.requests[userKey] = reqWindow
	}

	// Update last access time
	reqWindow.lastAccess = now

	// Remove requests outside the time window
	var validRequests []time.Time
	for _, req := range reqWindow.times {
		if now.Sub(req) < window {
			validRequests = append(validRequests, req)
		}
	}

	// Check if we're within the dynamic limit
	if len(validRequests) >= limit {
		reqWindow.times = validRequests
		return false
	}

	// Add this request
	validRequests = append(validRequests, now)
	reqWindow.times = validRequests

	return true
}
