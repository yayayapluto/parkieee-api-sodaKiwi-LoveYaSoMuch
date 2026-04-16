package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("GATE_JWT_SECRET", "gate-secret")
	t.Setenv("S3_ENDPOINT", "https://example-s3.local")
	t.Setenv("S3_BUCKET", "bucket")
	t.Setenv("S3_ACCESS_KEY", "key")
	t.Setenv("S3_SECRET_KEY", "secret")
	t.Setenv("S3_PUBLIC_BASE_URL", "https://cdn.local")
	t.Setenv("MIDTRANS_SERVER_KEY", "mid-server")
	t.Setenv("MIDTRANS_CLIENT_KEY", "mid-client")
	t.Setenv("LPR_SERVICE_URL", "http://localhost:8001")
}

func TestLoad_UseCSRF_DefaultTrue(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("USE_CSRF", "")

	cfg, err := Load()
	require.NoError(t, err)
	assert.True(t, cfg.UseCSRF)
}

func TestLoad_UseCSRF_ExplicitFalse(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("USE_CSRF", "false")

	cfg, err := Load()
	require.NoError(t, err)
	assert.False(t, cfg.UseCSRF)
}

func TestLoad_UseCSRF_InvalidFallbackTrue(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("USE_CSRF", "not-a-bool")

	cfg, err := Load()
	require.NoError(t, err)
	assert.True(t, cfg.UseCSRF)
}
