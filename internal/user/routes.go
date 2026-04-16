package user

import "github.com/gofiber/fiber/v2"

import "github.com/yyypluto/parkieee-api/pkg/middleware"

func RegisterRoutes(router fiber.Router, h *Handler, jwtSecret string) {
	users := router.Group("/users", middleware.AuthMiddleware(jwtSecret), middleware.RequirePermission("user:manage"))
	users.Get("", h.ListUsers)
	users.Get("/", h.ListUsers)
	users.Post("", h.CreateUser)
	users.Post("/", h.CreateUser)
	users.Get("/:id", h.GetUser)
	users.Put("/:id", h.UpdateUser)
	users.Patch("/:id/deactivate", h.DeactivateUser)
	users.Patch("/:id/reset-password", h.ResetPassword)
}
