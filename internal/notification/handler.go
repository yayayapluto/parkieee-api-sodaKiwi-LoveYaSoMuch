package notification

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) List(c *fiber.Ctx) error {
	userIDStr, _ := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(401).JSON(errResponse("UNAUTHORIZED", "invalid user context"))
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, total, err := h.svc.List(c.Context(), &userID, page, limit)
	if err != nil {
		return c.Status(500).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.JSON(okResponse(fiber.Map{
		"items": rows, "total": total, "page": page, "limit": limit,
	}))
}

func (h *Handler) MarkRead(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(errResponse("VALIDATION_ERROR", "invalid notification id"))
	}
	if err := h.svc.MarkRead(c.Context(), id); err != nil {
		return c.Status(500).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.JSON(okResponse(fiber.Map{"id": id}))
}

func (h *Handler) MarkAllRead(c *fiber.Ctx) error {
	userIDStr, _ := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(401).JSON(errResponse("UNAUTHORIZED", "invalid user context"))
	}
	if err := h.svc.MarkAllRead(c.Context(), &userID); err != nil {
		return c.Status(500).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.JSON(okResponse(fiber.Map{"message": "all notifications marked as read"}))
}

func (h *Handler) UnreadCount(c *fiber.Ctx) error {
	userIDStr, _ := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(401).JSON(errResponse("UNAUTHORIZED", "invalid user context"))
	}

	total, err := h.svc.UnreadCount(c.Context(), &userID)
	if err != nil {
		return c.Status(500).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.JSON(okResponse(fiber.Map{"unread_count": total}))
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
