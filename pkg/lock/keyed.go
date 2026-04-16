// pkg/lock/keyed.go
package lock

import (
	"sync"
	"time"
)

type entry struct {
	expiresAt time.Time
}

type KeyedMutex struct {
	mu    sync.Mutex
	locks map[string]entry
}

func NewKeyedMutex() *KeyedMutex {
	km := &KeyedMutex{locks: make(map[string]entry)}
	go km.cleanup()
	return km
}

func (km *KeyedMutex) TryLock(key string, ttl time.Duration) bool {
	km.mu.Lock()
	defer km.mu.Unlock()
	if e, ok := km.locks[key]; ok && time.Now().Before(e.expiresAt) {
		return false
	}
	km.locks[key] = entry{expiresAt: time.Now().Add(ttl)}
	return true
}

func (km *KeyedMutex) Unlock(key string) {
	km.mu.Lock()
	delete(km.locks, key)
	km.mu.Unlock()
}

func (km *KeyedMutex) cleanup() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		now := time.Now()
		km.mu.Lock()
		for k, e := range km.locks {
			if now.After(e.expiresAt) {
				delete(km.locks, k)
			}
		}
		km.mu.Unlock()
	}
}
