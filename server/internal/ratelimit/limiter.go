package ratelimit

import (
	"sync"
	"time"
)

type Limiter struct {
	mu          sync.Mutex
	hosts       map[string]time.Time
	minInterval time.Duration
}

func New(minInterval time.Duration) *Limiter {
	return &Limiter{
		hosts:       make(map[string]time.Time),
		minInterval: minInterval,
	}
}

func (l *Limiter) Wait(host string) {
	l.mu.Lock()
	lastRequest, exists := l.hosts[host]
	l.mu.Unlock()

	if exists {
		elapsed := time.Since(lastRequest)
		if elapsed < l.minInterval {
			time.Sleep(l.minInterval - elapsed)
		}
	}

	l.mu.Lock()
	l.hosts[host] = time.Now()
	l.mu.Unlock()
}

func (l *Limiter) Allow(host string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	lastRequest, exists := l.hosts[host]
	if !exists {
		l.hosts[host] = time.Now()
		return true
	}

	if time.Since(lastRequest) >= l.minInterval {
		l.hosts[host] = time.Now()
		return true
	}

	return false
}

func (l *Limiter) Reset(host string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hosts, host)
}

func (l *Limiter) ResetAll() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hosts = make(map[string]time.Time)
}

// Ensure Limiter implements RateLimiter interface
var _ RateLimiter = (*Limiter)(nil)
