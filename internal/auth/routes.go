// internal/auth/routes.go
package auth

import "github.com/gofiber/fiber/v2"

import "github.com/yyypluto/parkieee-api/pkg/middleware"

func RegisterRoutes(router fiber.Router, h *Handler, jwtSecret string) {
	auth := router.Group("/auth")
	auth.Post("/login", h.Login)
	auth.Post("/refresh", h.Refresh)
	auth.Post("/logout", h.Logout)
	auth.Get("/me", middleware.AuthMiddleware(jwtSecret), h.Me)
}
