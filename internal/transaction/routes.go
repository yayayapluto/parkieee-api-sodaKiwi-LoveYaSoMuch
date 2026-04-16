package transaction

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yyypluto/parkieee-api/pkg/middleware"
)

func RegisterRoutes(router fiber.Router, h *Handler, jwtSecret string) {
	read := router.Group("", middleware.AuthMiddleware(jwtSecret), middleware.RequirePermission("transaction:read"))
	write := router.Group("", middleware.AuthMiddleware(jwtSecret), middleware.RequirePermission("transaction:write"))

	write.Post("/transactions/entry", h.Entry)
	write.Post("/transactions/exit", h.Exit)
	write.Post("/uploads/entry-photo", h.UploadEntryPhoto)
	write.Post("/uploads/exit-photo", h.UploadExitPhoto)

	read.Get("/transactions", h.List)
	read.Get("/transactions/:id", h.Detail)
	read.Get("/transactions/:id/logs", h.Logs)
}
