package ocr

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Detect_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/detect-plate", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		payload, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var body map[string]string
		require.NoError(t, json.Unmarshal(payload, &body))
		assert.Equal(t, "/tmp/image.jpg", body["image_path"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"plate_text":"B1234CD","confidence":0.97,"image_url":"https://cdn/out.jpg"}`))
	}))
	defer ts.Close()

	c := NewLPRClient(ts.URL)
	out, err := c.Detect(context.Background(), "/tmp/image.jpg")
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "B1234CD", out.PlateText)
	assert.Equal(t, 0.97, out.Confidence)
	assert.Equal(t, "https://cdn/out.jpg", out.ImageURL)
}

func TestClient_Detect_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer ts.Close()

	c := NewLPRClient(ts.URL)
	out, err := c.Detect(context.Background(), "/tmp/image.jpg")
	assert.Nil(t, out)
	assert.Error(t, err)
}

func TestClient_Detect_RetriesTransientFailure(t *testing.T) {
	var callCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&callCount, 1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"plate_text":"B1234CD","confidence":0.97,"image_url":"https://cdn/out.jpg"}`))
	}))
	defer ts.Close()

	c := NewLPRClient(ts.URL)
	out, err := c.Detect(context.Background(), "/tmp/image.jpg")
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "B1234CD", out.PlateText)
	assert.Equal(t, int32(2), atomic.LoadInt32(&callCount))
}
