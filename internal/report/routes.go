package report

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yyypluto/parkieee-api/pkg/middleware"
)

func RegisterRoutes(router fiber.Router, h *Handler, jwtSecret string) {
	g := router.Group("/reports",
		middleware.AuthMiddleware(jwtSecret),
		middleware.RequirePermission("report:read"),
	)
	// Static export route BEFORE parametric patterns
	g.Get("/revenue/export", h.ExportRevenue)
	g.Get("/revenue", h.Revenue)
	g.Get("/occupancy", h.Occupancy)
}
