// pkg/ratelimit/limiter.go
package ratelimit

import (
	"sync"

	"golang.org/x/time/rate"
)

type Limiter struct {
	mu      sync.Mutex
	keys    map[string]*rate.Limiter
	rps     rate.Limit
	burst   int
}

func NewLimiter(rps, burst int) *Limiter {
	return &Limiter{
		keys:  make(map[string]*rate.Limiter),
		rps:   rate.Limit(rps),
		burst: burst,
	}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	lim, ok := l.keys[key]
	if !ok {
		lim = rate.NewLimiter(l.rps, l.burst)
		l.keys[key] = lim
	}
	l.mu.Unlock()
	return lim.Allow()
}
