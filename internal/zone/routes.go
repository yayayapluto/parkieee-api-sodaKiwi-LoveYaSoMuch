package zone

import "github.com/gofiber/fiber/v2"

import "github.com/yyypluto/parkieee-api/pkg/middleware"

func RegisterRoutes(router fiber.Router, h *Handler, jwtSecret string) {
	read := router.Group("", middleware.AuthMiddleware(jwtSecret), middleware.RequirePermission("zone:read"))
	write := router.Group("", middleware.AuthMiddleware(jwtSecret), middleware.RequirePermission("zone:write"))

	read.Get("/zones", h.ListZones)
	write.Post("/zones", h.CreateZone)
	read.Get("/zones/:id", h.GetZone)
	write.Put("/zones/:id", h.UpdateZone)

	read.Get("/gates", h.ListGates)
	write.Post("/gates", h.CreateGate)
	read.Get("/gates/:id", h.GetGate)
	write.Put("/gates/:id", h.UpdateGate)

	read.Get("/gates/:id/devices", h.ListGateDevices)
	write.Patch("/gate-devices/:id/status", h.UpdateGateDeviceStatus)

	// Pair request/verify is kiosk bootstrap path before authenticated gate channel is available.
	router.Post("/gates/pair/request", h.PairRequest)
	write.Post("/gates/pair/confirm", h.PairConfirm)
	router.Get("/gates/pair/verify/:code", h.PairVerify)
}
