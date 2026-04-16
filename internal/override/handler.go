package override

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Apply(c *fiber.Ctx) error {
	txID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid transaction id"))
	}
	var body struct {
		OverrideType string `json:"override_type"`
		Reason       string `json:"reason"`
		AdjustedFee  int    `json:"adjusted_fee"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}
	if body.Reason == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "reason is required"))
	}
	operatorIDStr, _ := c.Locals("user_id").(string)
	operatorID, err := uuid.Parse(operatorIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResponse("UNAUTHORIZED", "invalid user context"))
	}
	rec, err := h.svc.Apply(c.Context(), ApplyInput{
		TransactionID: txID,
		OperatorID:    operatorID,
		OverrideType:  body.OverrideType,
		Reason:        body.Reason,
		AdjustedFee:   body.AdjustedFee,
	})
	if errors.Is(err, ErrTransactionNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("NOT_FOUND", "transaction not found"))
	}
	if errors.Is(err, ErrDailyLimitExceeded) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(errResponse("LIMIT_EXCEEDED", "daily override limit exceeded"))
	}
	if errors.Is(err, ErrInvalidOverrideType) {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", err.Error()))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.JSON(okResponse(rec))
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
