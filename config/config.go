// config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port string

	// CORS
	CORSAllowedOrigins string

	// Database
	DatabaseURL string

	// JWT
	JWTSecret     string
	JWTExpMinutes int

	// Auth cookie
	CookieDomain   string
	CookieSecure   bool
	CookieSameSite string
	UseCSRF        bool

	// Gate JWT
	GateJWTSecret string

	// S3
	S3Endpoint      string
	S3Bucket        string
	S3AccessKey     string
	S3SecretKey     string
	S3Region        string
	S3PublicBaseURL string

	// Midtrans
	MidtransServerKey string
	MidtransClientKey string
	MidtransEnv       string // "sandbox" or "production"

	// LPR
	LPRServiceURL string

	// App
	TenantID string

	// Telegram Bot
	TelegramBotToken string
	TelegramAdminIDs string

	// Logging
	LogLevel  string // "debug" | "info" | "warn" | "error"
	LogFormat string // "json" | "text"
}

func Load() (*Config, error) {
	// Load .env if present (dev convenience; ignored in production)
	_ = godotenv.Load()

	cfg := &Config{
		Port:               getEnv("PORT", "8000"),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3001,http://localhost:5173"),
		DatabaseURL:        mustGetEnv("DATABASE_URL"),
		JWTSecret:          mustGetEnv("JWT_SECRET"),
		GateJWTSecret:      mustGetEnv("GATE_JWT_SECRET"),
		CookieDomain:       getEnv("COOKIE_DOMAIN", ""),
		CookieSecure:       getEnvBool("COOKIE_SECURE", false),
		CookieSameSite:     getEnv("COOKIE_SAME_SITE", "Strict"),
		UseCSRF:            getEnvBool("USE_CSRF", true),
		S3Endpoint:         mustGetEnv("S3_ENDPOINT"),
		S3Bucket:           mustGetEnv("S3_BUCKET"),
		S3AccessKey:        mustGetEnv("S3_ACCESS_KEY"),
		S3SecretKey:        mustGetEnv("S3_SECRET_KEY"),
		S3Region:           getEnv("S3_REGION", "us-east-1"),
		S3PublicBaseURL:    mustGetEnv("S3_PUBLIC_BASE_URL"),
		MidtransServerKey:  mustGetEnv("MIDTRANS_SERVER_KEY"),
		MidtransClientKey:  mustGetEnv("MIDTRANS_CLIENT_KEY"),
		MidtransEnv:        getEnv("MIDTRANS_ENV", "sandbox"),
		LPRServiceURL:      mustGetEnv("LPR_SERVICE_URL"),
		TenantID:           getEnv("TENANT_ID", "00000000-0000-0000-0000-000000000001"),
		TelegramBotToken:   getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramAdminIDs:   getEnv("TELEGRAM_ADMIN_IDS", ""),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		LogFormat:          getEnv("LOG_FORMAT", "text"),
	}

	var err error
	cfg.JWTExpMinutes, err = strconv.Atoi(getEnv("JWT_EXP_MINUTES", "15"))
	if err != nil {
		return nil, fmt.Errorf("JWT_EXP_MINUTES must be an integer: %w", err)
	}

	return cfg, nil
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required env var %s is not set", key))
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}

	return parsed
}
