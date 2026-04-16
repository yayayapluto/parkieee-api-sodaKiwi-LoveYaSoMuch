package resilience

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClientDo_RetryOnRetryableStatus(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&callCount, 1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.Client(), HTTPPolicy{
		Name:                       "test-retry",
		MaxRetries:                 2,
		BaseBackoff:                1 * time.Millisecond,
		MaxBackoff:                 2 * time.Millisecond,
		RequestTimeout:             1 * time.Second,
		BreakerConsecutiveFailures: 10,
		RetryableStatusCodes:       StatusSet(http.StatusServiceUnavailable),
	})

	resp, err := client.Do(context.Background(), func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), atomic.LoadInt32(&callCount))
	_ = resp.Body.Close()
}

func TestHTTPClientDo_BreakerOpensAfterConsecutiveFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewHTTPClient(server.Client(), HTTPPolicy{
		Name:                       "test-breaker",
		MaxRetries:                 0,
		BaseBackoff:                1 * time.Millisecond,
		MaxBackoff:                 2 * time.Millisecond,
		RequestTimeout:             1 * time.Second,
		BreakerConsecutiveFailures: 2,
		BreakerTimeout:             1 * time.Minute,
		RetryableStatusCodes:       StatusSet(http.StatusServiceUnavailable),
	})

	for i := 0; i < 2; i++ {
		_, err := client.Do(context.Background(), func(ctx context.Context) (*http.Request, error) {
			return http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		})
		require.Error(t, err)
	}

	_, err := client.Do(context.Background(), func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, gobreaker.ErrOpenState))
}

func TestHTTPClientDo_DoesNotRetryNonRetryableStatus(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewHTTPClient(server.Client(), HTTPPolicy{
		Name:                       "test-no-retry-400",
		MaxRetries:                 3,
		BaseBackoff:                1 * time.Millisecond,
		MaxBackoff:                 2 * time.Millisecond,
		RequestTimeout:             1 * time.Second,
		BreakerConsecutiveFailures: 10,
		RetryableStatusCodes:       StatusSet(http.StatusServiceUnavailable),
	})

	resp, err := client.Do(context.Background(), func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
	_ = resp.Body.Close()
}
