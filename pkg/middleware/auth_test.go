package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestGateTokenFromRequest_QueryTakesPrecedence(t *testing.T) {
	app := fiber.New()
	app.Get("/gate", func(c *fiber.Ctx) error {
		token := gateTokenFromRequest(c)
		return c.SendString(token)
	})

	req := httptest.NewRequest(http.MethodGet, "/gate?gate_token=abc123", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "abc123", string(body))
}

func TestExtractUserAccessToken_PrefersCookie(t *testing.T) {
	app := fiber.New()
	app.Get("/user-token", func(c *fiber.Ctx) error {
		return c.SendString(extractUserAccessToken(c))
	})

	req := httptest.NewRequest(http.MethodGet, "/user-token", nil)
	req.Header.Set("Authorization", "Bearer from-header")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "from-cookie"})

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "from-cookie", string(body))
}

func TestExtractUserAccessToken_FallsBackToBearer(t *testing.T) {
	app := fiber.New()
	app.Get("/user-token", func(c *fiber.Ctx) error {
		return c.SendString(extractUserAccessToken(c))
	})

	req := httptest.NewRequest(http.MethodGet, "/user-token", nil)
	req.Header.Set("Authorization", "Bearer from-header")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "from-header", string(body))
}

func TestCSRFMiddleware_AllowsSafeMethod(t *testing.T) {
	app := fiber.New()
	app.Use(CSRFMiddleware())
	app.Get("/api/v1/users", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestCSRFMiddleware_SkipsAuthLogin(t *testing.T) {
	app := fiber.New()
	app.Use(CSRFMiddleware())
	app.Post("/api/v1/auth/login", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestCSRFMiddleware_SkipsAuthRefresh(t *testing.T) {
	app := fiber.New()
	app.Use(CSRFMiddleware())
	app.Post("/api/v1/auth/refresh", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestCSRFMiddleware_SkipsAuthLogout(t *testing.T) {
	app := fiber.New()
	app.Use(CSRFMiddleware())
	app.Post("/api/v1/auth/logout", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestCSRFMiddleware_RejectsTokenMismatch(t *testing.T) {
	app := fiber.New()
	app.Use(CSRFMiddleware())
	app.Post("/api/v1/users", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	req.Header.Set("X-CSRF-Token", "token-a")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "token-b"})

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestCSRFMiddleware_AllowsMatchingTokens(t *testing.T) {
	app := fiber.New()
	app.Use(CSRFMiddleware())
	app.Post("/api/v1/users", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	req.Header.Set("X-CSRF-Token", "token-a")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "token-a"})

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}
