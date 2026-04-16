package payment

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yyypluto/parkieee-api/pkg/middleware"
)

func RegisterRoutes(router fiber.Router, h *Handler, jwtSecret string) {
	write := router.Group("", middleware.AuthMiddleware(jwtSecret), middleware.RequirePermission("transaction:write"))
	read := router.Group("", middleware.AuthMiddleware(jwtSecret), middleware.RequirePermission("transaction:read"))

	write.Post("/payments/cash", h.Cash)
	write.Post("/payments/qris", h.QRIS)
	write.Post("/payments/refunds", h.Refund)

	router.Post("/payments/midtrans/callback", h.MidtransCallback)
	read.Get("/payments/:transaction_id", h.GetByTransaction)
	read.Get("/payments/:transaction_id/refunds", h.ListRefunds)
}
