package fee

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type feeConfigRequest struct {
	ZoneID             string  `json:"zone_id"`
	VehicleTypeID      string  `json:"vehicle_type_id"`
	BaseFee            int     `json:"base_fee"`
	GracePeriodMinutes int     `json:"grace_period_minutes"`
	IsActive           bool    `json:"is_active"`
	EffectiveFrom      string  `json:"effective_from"`
	EffectiveUntil     *string `json:"effective_until"`
}

type holidayRateRequest struct {
	Name          string  `json:"name"`
	HolidayDate   string  `json:"holiday_date"`
	AdditionalFee int     `json:"additional_fee"`
	Multiplier    float64 `json:"multiplier"`
	IsActive      bool    `json:"is_active"`
}

func (h *Handler) ListFeeConfigs(c *fiber.Ctx) error {
	rows, err := h.svc.ListFeeConfigs()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.JSON(okResponse(rows))
}

func (h *Handler) CreateFeeConfig(c *fiber.Ctx) error {
	var body feeConfigRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}
	in, err := parseFeeConfigInput(body)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", err.Error()))
	}
	out, err := h.svc.CreateFeeConfig(in)
	if err != nil {
		return mapFeeError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(okResponse(out))
}

func (h *Handler) GetFeeConfig(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid fee config id"))
	}
	out, err := h.svc.GetFeeConfig(id)
	if err != nil {
		return mapFeeError(c, err)
	}
	return c.JSON(okResponse(out))
}

func (h *Handler) UpdateFeeConfig(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid fee config id"))
	}
	var body feeConfigRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}
	in, err := parseFeeConfigInput(body)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", err.Error()))
	}
	out, err := h.svc.UpdateFeeConfig(id, in)
	if err != nil {
		return mapFeeError(c, err)
	}
	return c.JSON(okResponse(out))
}

func (h *Handler) DeleteFeeConfig(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid fee config id"))
	}
	if err := h.svc.DeleteFeeConfig(id); err != nil {
		return mapFeeError(c, err)
	}
	return c.JSON(okResponse(fiber.Map{"id": id, "deleted": true}))
}

func (h *Handler) ListFeeTiers(c *fiber.Ctx) error {
	feeConfigID, err := parseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid fee config id"))
	}
	rows, err := h.svc.ListFeeTiers(feeConfigID)
	if err != nil {
		return mapFeeError(c, err)
	}
	return c.JSON(okResponse(rows))
}

func (h *Handler) CreateFeeTier(c *fiber.Ctx) error {
	feeConfigID, err := parseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid fee config id"))
	}
	var body struct {
		TierOrder       int  `json:"tier_order"`
		DurationMinutes int  `json:"duration_minutes"`
		FeeAmount       int  `json:"fee_amount"`
		IsLastTier      bool `json:"is_last_tier"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}
	out, err := h.svc.CreateFeeTier(feeConfigID, FeeTierInput{
		TierOrder:       body.TierOrder,
		DurationMinutes: body.DurationMinutes,
		FeeAmount:       body.FeeAmount,
		IsLastTier:      body.IsLastTier,
	})
	if err != nil {
		return mapFeeError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(okResponse(out))
}

func (h *Handler) UpdateFeeTier(c *fiber.Ctx) error {
	feeConfigID, err := parseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid fee config id"))
	}
	tierID, err := parseUUIDParam(c, "tier_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid fee tier id"))
	}
	var body struct {
		TierOrder       int  `json:"tier_order"`
		DurationMinutes int  `json:"duration_minutes"`
		FeeAmount       int  `json:"fee_amount"`
		IsLastTier      bool `json:"is_last_tier"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}
	out, err := h.svc.UpdateFeeTier(feeConfigID, tierID, FeeTierInput{
		TierOrder:       body.TierOrder,
		DurationMinutes: body.DurationMinutes,
		FeeAmount:       body.FeeAmount,
		IsLastTier:      body.IsLastTier,
	})
	if err != nil {
		return mapFeeError(c, err)
	}
	return c.JSON(okResponse(out))
}

func (h *Handler) DeleteFeeTier(c *fiber.Ctx) error {
	feeConfigID, err := parseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid fee config id"))
	}
	tierID, err := parseUUIDParam(c, "tier_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid fee tier id"))
	}
	if err := h.svc.DeleteFeeTier(feeConfigID, tierID); err != nil {
		return mapFeeError(c, err)
	}
	return c.JSON(okResponse(fiber.Map{"id": tierID, "deleted": true}))
}

func (h *Handler) ListHolidayRates(c *fiber.Ctx) error {
	rows, err := h.svc.ListHolidayRates()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.JSON(okResponse(rows))
}

func (h *Handler) CreateHolidayRate(c *fiber.Ctx) error {
	var body holidayRateRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}
	in, err := parseHolidayRateInput(body)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", err.Error()))
	}
	out, err := h.svc.CreateHolidayRate(in)
	if err != nil {
		return mapFeeError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(okResponse(out))
}

func (h *Handler) GetHolidayRate(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid holiday rate id"))
	}
	out, err := h.svc.GetHolidayRate(id)
	if err != nil {
		return mapFeeError(c, err)
	}
	return c.JSON(okResponse(out))
}

func (h *Handler) UpdateHolidayRate(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid holiday rate id"))
	}
	var body holidayRateRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}
	in, err := parseHolidayRateInput(body)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", err.Error()))
	}
	out, err := h.svc.UpdateHolidayRate(id, in)
	if err != nil {
		return mapFeeError(c, err)
	}
	return c.JSON(okResponse(out))
}

func (h *Handler) DeleteHolidayRate(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid holiday rate id"))
	}
	if err := h.svc.DeleteHolidayRate(id); err != nil {
		return mapFeeError(c, err)
	}
	return c.JSON(okResponse(fiber.Map{"id": id, "deleted": true}))
}

func (h *Handler) ListSystemConfigs(c *fiber.Ctx) error {
	rows, err := h.svc.ListSystemConfigs()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.JSON(okResponse(rows))
}

func (h *Handler) UpdateSystemConfig(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "key required"))
	}
	var body struct {
		Value string `json:"value"`
	}
	if err := c.BodyParser(&body); err != nil || body.Value == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "value required"))
	}
	if err := h.svc.UpdateSystemConfig(key, body.Value); err != nil {
		return mapFeeError(c, err)
	}
	return c.JSON(okResponse(fiber.Map{"key": key, "value": body.Value}))
}

func mapFeeError(c *fiber.Ctx, err error) error {
	if errors.Is(err, ErrValidation) {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", err.Error()))
	}
	if errors.Is(err, ErrFeeConfigNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("FEE_CONFIG_NOT_FOUND", err.Error()))
	}
	if errors.Is(err, ErrFeeTierNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("FEE_TIER_NOT_FOUND", err.Error()))
	}
	if errors.Is(err, ErrHolidayRateNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("HOLIDAY_RATE_NOT_FOUND", err.Error()))
	}
	return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
}

func parseUUIDParam(c *fiber.Ctx, key string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(c.Params(key)))
}

func parseDateOnly(value string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(value))
}

func parseFeeConfigInput(body feeConfigRequest) (FeeConfigInput, error) {
	zoneID, err := uuid.Parse(strings.TrimSpace(body.ZoneID))
	if err != nil {
		return FeeConfigInput{}, errors.New("zone_id must be valid UUID")
	}
	vehicleTypeID, err := uuid.Parse(strings.TrimSpace(body.VehicleTypeID))
	if err != nil {
		return FeeConfigInput{}, errors.New("vehicle_type_id must be valid UUID")
	}
	effectiveFrom, err := time.Parse(time.RFC3339, strings.TrimSpace(body.EffectiveFrom))
	if err != nil {
		return FeeConfigInput{}, errors.New("effective_from must be RFC3339")
	}
	var effectiveUntil *time.Time
	if body.EffectiveUntil != nil && strings.TrimSpace(*body.EffectiveUntil) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*body.EffectiveUntil))
		if err != nil {
			return FeeConfigInput{}, errors.New("effective_until must be RFC3339")
		}
		effectiveUntil = &t
	}
	return FeeConfigInput{
		ZoneID:             zoneID,
		VehicleTypeID:      vehicleTypeID,
		BaseFee:            body.BaseFee,
		GracePeriodMinutes: body.GracePeriodMinutes,
		IsActive:           body.IsActive,
		EffectiveFrom:      effectiveFrom,
		EffectiveUntil:     effectiveUntil,
	}, nil
}

func parseHolidayRateInput(body holidayRateRequest) (HolidayRateInput, error) {
	holidayDate, err := parseDateOnly(body.HolidayDate)
	if err != nil {
		return HolidayRateInput{}, errors.New("holiday_date must be YYYY-MM-DD")
	}
	return HolidayRateInput{
		Name:          strings.TrimSpace(body.Name),
		HolidayDate:   holidayDate,
		AdditionalFee: body.AdditionalFee,
		Multiplier:    body.Multiplier,
		IsActive:      body.IsActive,
	}, nil
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
