package fee

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yyypluto/parkieee-api/pkg/middleware"
)

func RegisterRoutes(router fiber.Router, h *Handler, jwtSecret string) {
	read := router.Group("",
		middleware.AuthMiddleware(jwtSecret),
		middleware.RequirePermission("fee:read"),
	)
	write := router.Group("",
		middleware.AuthMiddleware(jwtSecret),
		middleware.RequirePermission("fee:write"),
	)

	read.Get("/fee-configs", h.ListFeeConfigs)
	write.Post("/fee-configs", h.CreateFeeConfig)
	read.Get("/fee-configs/:id", h.GetFeeConfig)
	write.Put("/fee-configs/:id", h.UpdateFeeConfig)
	write.Delete("/fee-configs/:id", h.DeleteFeeConfig)

	read.Get("/fee-configs/:id/tiers", h.ListFeeTiers)
	write.Post("/fee-configs/:id/tiers", h.CreateFeeTier)
	write.Put("/fee-configs/:id/tiers/:tier_id", h.UpdateFeeTier)
	write.Delete("/fee-configs/:id/tiers/:tier_id", h.DeleteFeeTier)

	read.Get("/holiday-rates", h.ListHolidayRates)
	write.Post("/holiday-rates", h.CreateHolidayRate)
	read.Get("/holiday-rates/:id", h.GetHolidayRate)
	write.Put("/holiday-rates/:id", h.UpdateHolidayRate)
	write.Delete("/holiday-rates/:id", h.DeleteHolidayRate)

	system := router.Group("/system-configs",
		middleware.AuthMiddleware(jwtSecret),
		middleware.RequirePermission("user:manage"),
	)
	system.Get("", h.ListSystemConfigs)
	system.Put("/:key", h.UpdateSystemConfig)
}
