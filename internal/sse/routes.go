package sse

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yyypluto/parkieee-api/pkg/middleware"
	"gorm.io/gorm"
)

func RegisterRoutes(router fiber.Router, h *Handler, db *gorm.DB, jwtSecret string) {
	group := router.Group("/sse")
	group.Get("/gate/:id", middleware.GateAuthMiddleware(db), h.GateEvents)
	group.Get("/cashier", middleware.AuthMiddleware(jwtSecret), middleware.RequirePermission("transaction:read"), h.CashierEvents)
}
