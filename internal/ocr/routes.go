package ocr

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yyypluto/parkieee-api/pkg/middleware"
)

func RegisterRoutes(router fiber.Router, h *Handler, jwtSecret string) {
	read := router.Group("", middleware.AuthMiddleware(jwtSecret), middleware.RequirePermission("transaction:read"))
	write := router.Group("", middleware.AuthMiddleware(jwtSecret), middleware.RequirePermission("transaction:write"))
	read.Get("/ocr/jobs", h.List)
	write.Post("/ocr/jobs/:id/review", h.Review)
}
