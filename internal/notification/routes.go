package notification

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yyypluto/parkieee-api/pkg/middleware"
)

func RegisterRoutes(router fiber.Router, h *Handler, jwtSecret string) {
	g := router.Group("/notifications",
		middleware.AuthMiddleware(jwtSecret),
		middleware.RequirePermission("notification:read"),
	)
	g.Get("/unread-count", h.UnreadCount)
	g.Get("", h.List)
	g.Patch("/read-all", h.MarkAllRead)
	g.Patch("/:id/read", h.MarkRead)
}
