package ratelimit

// RateLimiter defines the interface for rate limiting backends
// This allows for both in-memory (single instance) and distributed (Redis) implementations
type RateLimiter interface {
	// Allow checks if a request from the given key is allowed
	// Returns true if allowed, false if rate limited
	Allow(key string) bool
}
