package report

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Revenue(c *fiber.Ctx) error {
	f := revenueFilterFromStrings(
		c.Query("from"), c.Query("to"),
		c.Query("zone_id"), c.Query("vehicle_type_id"),
	)
	rows, err := h.svc.GetRevenue(c.Context(), f)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.JSON(okResponse(fiber.Map{"items": rows, "count": len(rows)}))
}

func (h *Handler) Occupancy(c *fiber.Ctx) error {
	rows, err := h.svc.GetOccupancy(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.JSON(okResponse(fiber.Map{"items": rows}))
}

func (h *Handler) ExportRevenue(c *fiber.Ctx) error {
	f := revenueFilterFromStrings(
		c.Query("from"), c.Query("to"),
		c.Query("zone_id"), c.Query("vehicle_type_id"),
	)
	data, err := h.svc.ExportRevenueCSV(c.Context(), f)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	filename := fmt.Sprintf("revenue_%s.csv", time.Now().Format("2006-01-02"))
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	return c.Send(data)
}

func okResponse(data any) fiber.Map {
	return fiber.Map{"success": true, "data": data, "meta": fiber.Map{}, "error": nil}
}

func errResponse(code, message string) fiber.Map {
	return fiber.Map{
		"success": false, "data": nil, "meta": fiber.Map{},
		"error": fiber.Map{"code": code, "message": message},
	}
}
