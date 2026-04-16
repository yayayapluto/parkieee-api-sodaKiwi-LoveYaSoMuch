package override

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yyypluto/parkieee-api/pkg/middleware"
)

func RegisterRoutes(router fiber.Router, h *Handler, jwtSecret string) {
	g := router.Group("/transactions",
		middleware.AuthMiddleware(jwtSecret),
		middleware.RequirePermission("override:write"),
	)
	g.Post("/:id/override", h.Apply)
}
