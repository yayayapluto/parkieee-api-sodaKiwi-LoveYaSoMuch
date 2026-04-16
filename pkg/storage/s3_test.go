package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientPutObject_RetriesTransientFailure(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Contains(t, r.URL.Path, "/bucket/")
		if atomic.AddInt32(&callCount, 1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		"bucket",
		"access",
		"secret",
		"us-east-1",
		"https://cdn.local",
	)

	url, err := client.PutObject(context.Background(), "photos/test.jpg", []byte("img"), "image/jpeg")
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.local/photos/test.jpg", url)
	assert.Equal(t, int32(2), atomic.LoadInt32(&callCount))
}

func TestClientPutObject_DoesNotRetryNonRetryableStatus(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		"bucket",
		"access",
		"secret",
		"us-east-1",
		"https://cdn.local",
	)

	_, err := client.PutObject(context.Background(), "photos/test.jpg", []byte("img"), "image/jpeg")
	require.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
}
