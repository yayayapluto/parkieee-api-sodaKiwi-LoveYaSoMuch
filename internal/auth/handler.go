// internal/auth/handler.go
package auth

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

const (
	accessTokenCookieName       = "access_token"
	refreshTokenCookieName      = "refresh_token"
	csrfTokenCookieName         = "csrf_token"
	refreshTokenMaxAgeInSeconds = 7 * 24 * 60 * 60
)

type CookieConfig struct {
	Domain   string
	Secure   bool
	SameSite string
}

type Handler struct {
	svc          *Service
	cookieConfig CookieConfig
}

func NewHandler(svc *Service, cookieConfig CookieConfig) *Handler {
	if strings.TrimSpace(cookieConfig.SameSite) == "" {
		cookieConfig.SameSite = "Strict"
	}

	return &Handler{svc: svc, cookieConfig: cookieConfig}
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}
	if body.Username == "" || body.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "username and password are required"))
	}

	ip := c.IP()
	ua := string(c.Request().Header.UserAgent())

	result, err := h.svc.Login(body.Username, body.Password, ip, ua)
	if err == ErrInvalidCredentials {
		return c.Status(fiber.StatusUnauthorized).JSON(errResponse("INVALID_CREDENTIALS", "invalid username or password"))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	h.setAuthCookies(c, result.AccessToken, result.RefreshToken, result.CsrfToken, result.ExpiresAt)

	return c.JSON(okResponse(fiber.Map{
		"user": fiber.Map{
			"id":          result.UserID,
			"role":        result.Role,
			"permissions": result.Permissions,
			"session_id":  result.SessionID,
		},
		"expires_at": result.ExpiresAt,
	}))
}

func (h *Handler) Refresh(c *fiber.Ctx) error {
	rawRefreshToken := refreshTokenFromRequest(c)
	if rawRefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "refresh_token required"))
	}

	ua := string(c.Request().Header.UserAgent())
	result, err := h.svc.Refresh(rawRefreshToken, c.IP(), ua)
	if err == ErrTokenInvalid {
		return c.Status(fiber.StatusUnauthorized).JSON(errResponse("TOKEN_EXPIRED", "refresh token invalid or expired"))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	h.setAuthCookies(c, result.AccessToken, result.RefreshToken, result.CsrfToken, result.ExpiresAt)

	return c.JSON(okResponse(fiber.Map{
		"expires_at": result.ExpiresAt,
	}))
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	rawRefreshToken := refreshTokenFromRequest(c)
	if rawRefreshToken != "" {
		_ = h.svc.Logout(rawRefreshToken)
	}

	h.clearAuthCookies(c)

	return c.JSON(okResponse(fiber.Map{"message": "logged out"}))
}

func (h *Handler) Me(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(errResponse("TOKEN_EXPIRED", "invalid token claims"))
	}

	sessionID, _ := c.Locals("session_id").(string)

	currentUser, err := h.svc.GetCurrentUser(userID, sessionID)
	if err == ErrTokenInvalid || err == ErrUserNotFound {
		return c.Status(fiber.StatusUnauthorized).JSON(errResponse("TOKEN_EXPIRED", "invalid or expired token"))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.JSON(okResponse(fiber.Map{
		"user": fiber.Map{
			"id":          currentUser.ID,
			"role":        currentUser.Role,
			"permissions": currentUser.Permissions,
			"session_id":  currentUser.SessionID,
		},
	}))
}

func (h *Handler) setAuthCookies(c *fiber.Ctx, accessToken, refreshToken, csrfToken string, accessTokenExpiresAt time.Time) {
	accessTokenMaxAge := int(time.Until(accessTokenExpiresAt).Seconds())
	if accessTokenMaxAge < 0 {
		accessTokenMaxAge = 0
	}

	h.setCookie(c, accessTokenCookieName, accessToken, accessTokenMaxAge, true)
	h.setCookie(c, refreshTokenCookieName, refreshToken, refreshTokenMaxAgeInSeconds, true)
	h.setCookie(c, csrfTokenCookieName, csrfToken, accessTokenMaxAge, false)
}

func (h *Handler) clearAuthCookies(c *fiber.Ctx) {
	h.clearCookie(c, accessTokenCookieName, true)
	h.clearCookie(c, refreshTokenCookieName, true)
	h.clearCookie(c, csrfTokenCookieName, false)
}

func (h *Handler) setCookie(c *fiber.Ctx, name, value string, maxAgeInSeconds int, httpOnly bool) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   h.cookieConfig.Domain,
		MaxAge:   maxAgeInSeconds,
		Secure:   h.cookieConfig.Secure,
		HTTPOnly: httpOnly,
		SameSite: h.cookieConfig.SameSite,
	})
}

func (h *Handler) clearCookie(c *fiber.Ctx, name string, httpOnly bool) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Domain:   h.cookieConfig.Domain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Secure:   h.cookieConfig.Secure,
		HTTPOnly: httpOnly,
		SameSite: h.cookieConfig.SameSite,
	})
}

func refreshTokenFromRequest(c *fiber.Ctx) string {
	if token := strings.TrimSpace(c.Cookies(refreshTokenCookieName)); token != "" {
		return token
	}

	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&body); err != nil {
		return ""
	}

	return strings.TrimSpace(body.RefreshToken)
}

// Shared envelope helpers used across all handlers.
func okResponse(data any) fiber.Map {
	return fiber.Map{"success": true, "data": data, "meta": fiber.Map{}, "error": nil}
}

func errResponse(code, message string) fiber.Map {
	return fiber.Map{
		"success": false, "data": nil, "meta": fiber.Map{},
		"error": fiber.Map{"code": code, "message": message},
	}
}
