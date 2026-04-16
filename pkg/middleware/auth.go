// pkg/middleware/auth.go
package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/yyypluto/parkieee-api/pkg/ratelimit"
)

// RequestID attaches a unique X-Request-ID to every request.
func RequestID() fiber.Handler {
	return requestid.New()
}

// AuthMiddleware validates the human-user JWT.
// Sets c.Locals("user_id"), c.Locals("role"), c.Locals("permissions"), c.Locals("session_id").
func AuthMiddleware(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := extractUserAccessToken(c)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(errResponse("TOKEN_EXPIRED", "missing access token"))
		}

		claims := &jwtClaims{}
		t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !t.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(errResponse("TOKEN_EXPIRED", "invalid or expired token"))
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("role", claims.Role)
		c.Locals("permissions", claims.Permissions)
		c.Locals("session_id", claims.SessionID)
		return c.Next()
	}
}

// CSRFMiddleware validates the double-submit token for mutating HTTP methods.
func CSRFMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
			return c.Next()
		}

		path := strings.TrimRight(c.Path(), "/")
		if strings.HasSuffix(path, "/auth/login") ||
			strings.HasSuffix(path, "/auth/refresh") ||
			strings.HasSuffix(path, "/auth/logout") ||
			strings.HasSuffix(path, "/payments/midtrans/callback") {
			return c.Next()
		}

		headerToken := strings.TrimSpace(c.Get("X-CSRF-Token"))
		cookieToken := strings.TrimSpace(c.Cookies("csrf_token"))
		if headerToken == "" || cookieToken == "" || headerToken != cookieToken {
			return c.Status(fiber.StatusForbidden).JSON(errResponse("CSRF_INVALID", "invalid csrf token"))
		}

		return c.Next()
	}
}

// GateAuthMiddleware validates the gate_token by looking it up in the DB.
// Sets c.Locals("gate_id").
func GateAuthMiddleware(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := gateTokenFromRequest(c)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(errResponse("TOKEN_EXPIRED", "missing gate token"))
		}

		var gate struct {
			ID       uuid.UUID
			IsActive bool
		}
		err := db.Table("gates").Select("id, is_active").Where("gate_token = ?", token).First(&gate).Error
		if err != nil || !gate.IsActive {
			return c.Status(fiber.StatusUnauthorized).JSON(errResponse("PERMISSION_DENIED", "invalid gate token"))
		}

		c.Locals("gate_id", gate.ID.String())
		return c.Next()
	}
}

// RequirePermission checks that the user's permission list contains the given node.
func RequirePermission(node string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		perms, ok := c.Locals("permissions").([]string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(errResponse("PERMISSION_DENIED", "no permissions"))
		}
		for _, p := range perms {
			if p == node {
				return c.Next()
			}
		}
		return c.Status(fiber.StatusForbidden).JSON(errResponse("PERMISSION_DENIED", "missing permission: "+node))
	}
}

// RateLimiter returns a middleware that limits requests per key using an in-memory token bucket.
// keyFn extracts the rate-limit key from the request (e.g. IP or gate_id).
func RateLimiter(limiter *ratelimit.Limiter, keyFn func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := keyFn(c)
		if !limiter.Allow(key) {
			return c.Status(fiber.StatusTooManyRequests).JSON(errResponse("RATE_LIMIT_EXCEEDED", "too many requests"))
		}
		return c.Next()
	}
}

// jwtClaims mirrors auth.Claims without creating a circular import.
type jwtClaims struct {
	jwt.RegisteredClaims
	UserID      string   `json:"user_id"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	SessionID   string   `json:"session_id"`
}

func extractBearer(c *fiber.Ctx) string {
	h := c.Get("Authorization")
	if after, ok := strings.CutPrefix(h, "Bearer "); ok {
		return after
	}
	return ""
}

func extractUserAccessToken(c *fiber.Ctx) string {
	if token := strings.TrimSpace(c.Cookies("access_token")); token != "" {
		return token
	}

	return extractBearer(c)
}

func gateTokenFromRequest(c *fiber.Ctx) string {
	if token := c.Query("gate_token"); token != "" {
		return token
	}
	return extractBearer(c)
}

func errResponse(code, message string) fiber.Map {
	return fiber.Map{
		"success": false, "data": nil, "meta": fiber.Map{},
		"error": fiber.Map{"code": code, "message": message},
	}
}
