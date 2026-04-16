package audit

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func okResponse(data any) fiber.Map {
	return fiber.Map{"success": true, "data": data, "meta": fiber.Map{}, "error": nil}
}

func errResponse(code, message string) fiber.Map {
	return fiber.Map{
		"success": false, "data": nil, "meta": fiber.Map{},
		"error": fiber.Map{"code": code, "message": message},
	}
}

// List handles GET /audit-logs?q=&before=&limit=
func (h *Handler) List(c *fiber.Ctx) error {
	q := c.Query("q")

	var before time.Time
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(
				errResponse("BAD_REQUEST", "before must be RFC3339 (e.g. 2006-01-02T15:04:05Z)"),
			)
		}
		before = t
	}

	limit := c.QueryInt("limit", 50)

	logs, err := h.svc.Search(c.Context(), q, before, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			errResponse("INTERNAL_ERROR", err.Error()),
		)
	}
	return c.JSON(okResponse(logs))
}
