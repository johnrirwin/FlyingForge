package ratelimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLimiter is a distributed rate limiter backed by Redis
type RedisLimiter struct {
	client      *redis.Client
	prefix      string
	minInterval time.Duration
}

// NewRedis creates a new Redis-backed rate limiter
func NewRedis(client *redis.Client, prefix string, minInterval time.Duration) *RedisLimiter {
	if prefix == "" {
		prefix = "ratelimit:"
	}
	return &RedisLimiter{
		client:      client,
		prefix:      prefix,
		minInterval: minInterval,
	}
}

func (l *RedisLimiter) key(host string) string {
	return l.prefix + host
}

// Allow checks if a request from the given host is allowed
// Returns true if allowed, false if rate limited
func (l *RedisLimiter) Allow(host string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := l.key(host)

	// Use SET NX with expiration - atomic operation
	// If key doesn't exist, set it and return true (allowed)
	// If key exists, return false (rate limited)
	set, err := l.client.SetNX(ctx, key, time.Now().Unix(), l.minInterval).Result()
	if err != nil {
		// On Redis error, fail open (allow the request)
		return true
	}

	return set
}

// TimeUntilAllowed returns how long until the host can make another request
func (l *RedisLimiter) TimeUntilAllowed(host string) time.Duration {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := l.key(host)
	ttl, err := l.client.TTL(ctx, key).Result()
	if err != nil || ttl < 0 {
		return 0
	}
	return ttl
}

// Reset removes the rate limit for a host
func (l *RedisLimiter) Reset(host string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	l.client.Del(ctx, l.key(host))
}

// Ensure RedisLimiter implements RateLimiter interface
var _ RateLimiter = (*RedisLimiter)(nil)
