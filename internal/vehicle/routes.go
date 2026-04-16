package vehicle

import "github.com/gofiber/fiber/v2"

import "github.com/yyypluto/parkieee-api/pkg/middleware"

func RegisterRoutes(router fiber.Router, h *Handler, jwtSecret string) {
	read := router.Group("", middleware.AuthMiddleware(jwtSecret), middleware.RequirePermission("zone:read"))
	write := router.Group("", middleware.AuthMiddleware(jwtSecret), middleware.RequirePermission("zone:write"))

	read.Get("/vehicle-types", h.ListVehicleTypes)
	write.Post("/vehicle-types", h.CreateVehicleType)
	write.Put("/vehicle-types/:id", h.UpdateVehicleType)
	write.Delete("/vehicle-types/:id", h.DeleteVehicleType)

	read.Get("/vehicles", h.ListVehicles)
	write.Post("/vehicles", h.CreateVehicle)
	read.Get("/vehicles/:id", h.GetVehicle)
	write.Put("/vehicles/:id", h.UpdateVehicle)

	read.Get("/vehicles/:id/rfid-cards", h.ListRFIDCards)
	write.Post("/vehicles/:id/rfid-cards", h.AssignRFID)
	write.Patch("/rfid-cards/:id/deactivate", h.DeactivateRFID)
}
