package audit

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yyypluto/parkieee-api/pkg/middleware"
)

func RegisterRoutes(router fiber.Router, h *Handler, jwtSecret string) {
	g := router.Group("/audit-logs",
		middleware.AuthMiddleware(jwtSecret),
		middleware.RequirePermission("audit:read"),
	)
	g.Get("/", h.List)
}
