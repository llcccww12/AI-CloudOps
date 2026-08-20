package utils

import (
	"sync"
	"time"
)

// IPRateLimiter 简易内存 IP 限流（固定窗口）
type IPRateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	requests map[string][]time.Time
}

func NewIPRateLimiter(limit int, window time.Duration) *IPRateLimiter {
	if limit <= 0 {
		limit = 10
	}
	if window <= 0 {
		window = time.Minute
	}
	return &IPRateLimiter{
		limit:    limit,
		window:   window,
		requests: make(map[string][]time.Time),
	}
}

func (l *IPRateLimiter) Allow(ip string) bool {
	if ip == "" {
		ip = "unknown"
	}
	now := time.Now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	times := l.requests[ip]
	kept := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.limit {
		l.requests[ip] = kept
		return false
	}
	kept = append(kept, now)
	l.requests[ip] = kept
	return true
}
