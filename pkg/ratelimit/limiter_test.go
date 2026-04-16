// pkg/ratelimit/limiter_test.go
package ratelimit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yyypluto/parkieee-api/pkg/ratelimit"
)

func TestLimiter_AllowsUpToBurst(t *testing.T) {
	lim := ratelimit.NewLimiter(10, 3) // 10 req/s, burst 3
	assert.True(t, lim.Allow("key1"))
	assert.True(t, lim.Allow("key1"))
	assert.True(t, lim.Allow("key1"))
	// 4th request exceeds burst
	assert.False(t, lim.Allow("key1"))
}

func TestLimiter_SeparateKeysAreIndependent(t *testing.T) {
	lim := ratelimit.NewLimiter(10, 1)
	assert.True(t, lim.Allow("key-a"))
	assert.False(t, lim.Allow("key-a"))
	// key-b is fresh
	assert.True(t, lim.Allow("key-b"))
}
