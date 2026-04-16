// pkg/lock/keyed_test.go
package lock_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yyypluto/parkieee-api/pkg/lock"
)

func TestKeyedMutex_TryLockAndUnlock(t *testing.T) {
	km := lock.NewKeyedMutex()
	assert.True(t, km.TryLock("vehicle:1", time.Second))
	// Second lock on same key must fail
	assert.False(t, km.TryLock("vehicle:1", time.Second))
	km.Unlock("vehicle:1")
	// After unlock, must be acquirable again
	assert.True(t, km.TryLock("vehicle:1", time.Second))
	km.Unlock("vehicle:1")
}

func TestKeyedMutex_DifferentKeys(t *testing.T) {
	km := lock.NewKeyedMutex()
	assert.True(t, km.TryLock("vehicle:1", time.Second))
	assert.True(t, km.TryLock("vehicle:2", time.Second))
	km.Unlock("vehicle:1")
	km.Unlock("vehicle:2")
}

func TestKeyedMutex_TTLExpiry(t *testing.T) {
	km := lock.NewKeyedMutex()
	assert.True(t, km.TryLock("vehicle:3", 50*time.Millisecond))
	time.Sleep(100 * time.Millisecond)
	// TTL expired — must be re-acquirable
	assert.True(t, km.TryLock("vehicle:3", time.Second))
	km.Unlock("vehicle:3")
}
