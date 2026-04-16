// pkg/middleware/request_logger.go
package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yyypluto/parkieee-api/pkg/logger"
)

// RequestLogger logs each HTTP request after it completes.
// Fields: method, path, status, latency_ms, ip, request_id.
// Place after RequestID() so the X-Request-Id header is already set.
func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)

		requestID := c.GetRespHeader("X-Request-Id")
		status := c.Response().StatusCode()

		log := logger.Get()
		if requestID != "" {
			log = log.With("request_id", requestID)
		}

		attrs := []any{
			"method", c.Method(),
			"path", c.Path(),
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"ip", c.IP(),
		}

		if status >= 500 {
			log.Error("http request", attrs...)
		} else if status >= 400 {
			log.Warn("http request", attrs...)
		} else {
			log.Info("http request", attrs...)
		}

		return err
	}
}
