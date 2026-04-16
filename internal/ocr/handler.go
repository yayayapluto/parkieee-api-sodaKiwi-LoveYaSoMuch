package ocr

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, total, err := h.svc.ListJobs(c.UserContext(), c.Query("status"), page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.JSON(okResponse(fiber.Map{
		"items": rows,
		"total": total,
		"page":  page,
		"limit": limit,
	}))
}

func (h *Handler) Review(c *fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid OCR job id"))
	}

	var body struct {
		ManualPlate string `json:"manual_plate"`
		Match       bool   `json:"match"`
		ReviewNote  string `json:"review_note"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}

	reviewerID := userIDFromCtx(c)
	err = h.svc.ReviewJob(c.UserContext(), ReviewInput{
		JobID:       jobID,
		ManualPlate: body.ManualPlate,
		Match:       body.Match,
		ReviewNote:  body.ReviewNote,
		ReviewerID:  reviewerID,
	})
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(errResponse("OCR_JOB_NOT_FOUND", "OCR job not found"))
		}
		if errors.Is(err, ErrResultNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(errResponse("OCR_RESULT_NOT_FOUND", "OCR result not found"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.JSON(okResponse(fiber.Map{"review_submitted": true}))
}

func userIDFromCtx(c *fiber.Ctx) uuid.UUID {
	v, _ := c.Locals("user_id").(string)
	id, err := uuid.Parse(v)
	if err != nil {
		return uuid.Nil
	}
	return id
}

func okResponse(data any) fiber.Map {
	return fiber.Map{"success": true, "data": data, "meta": fiber.Map{}, "error": nil}
}

func errResponse(code, message string) fiber.Map {
	return fiber.Map{"success": false, "data": nil, "meta": fiber.Map{}, "error": fiber.Map{"code": code, "message": message}}
}
