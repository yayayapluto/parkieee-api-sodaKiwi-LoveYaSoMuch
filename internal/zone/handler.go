package zone

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListZones(c *fiber.Ctx) error {
	rows, err := h.svc.ListZones()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.JSON(okResponse(rows))
}

func (h *Handler) CreateZone(c *fiber.Ctx) error {
	var body struct {
		Name             string  `json:"name"`
		Description      string  `json:"description"`
		Capacity         int     `json:"capacity"`
		AdditionalFee    int     `json:"additional_fee"`
		ForVehicleTypeID *string `json:"for_vehicle_type_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}
	if strings.TrimSpace(body.Name) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "name is required"))
	}

	var vehicleTypeID *uuid.UUID
	if body.ForVehicleTypeID != nil && strings.TrimSpace(*body.ForVehicleTypeID) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*body.ForVehicleTypeID))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "for_vehicle_type_id must be a valid UUID"))
		}
		vehicleTypeID = &id
	}

	zone, err := h.svc.CreateZone(body.Name, body.Description, body.Capacity, body.AdditionalFee, vehicleTypeID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.Status(fiber.StatusCreated).JSON(okResponse(zone))
}

func (h *Handler) GetZone(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid zone id"))
	}
	out, err := h.svc.GetZone(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	if out == nil {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("ZONE_NOT_FOUND", "zone not found"))
	}
	return c.JSON(okResponse(out))
}

func (h *Handler) UpdateZone(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid zone id"))
	}

	var body struct {
		Name             string  `json:"name"`
		Description      string  `json:"description"`
		Capacity         int     `json:"capacity"`
		AdditionalFee    int     `json:"additional_fee"`
		ForVehicleTypeID *string `json:"for_vehicle_type_id"`
		IsActive         bool    `json:"is_active"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}

	var vehicleTypeID *uuid.UUID
	if body.ForVehicleTypeID != nil && strings.TrimSpace(*body.ForVehicleTypeID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(*body.ForVehicleTypeID))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "for_vehicle_type_id must be a valid UUID"))
		}
		vehicleTypeID = &parsed
	}

	out, err := h.svc.UpdateZone(id, body.Name, body.Description, body.Capacity, body.AdditionalFee, vehicleTypeID, body.IsActive)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	if out == nil {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("ZONE_NOT_FOUND", "zone not found"))
	}
	return c.JSON(okResponse(out))
}

func (h *Handler) ListGates(c *fiber.Ctx) error {
	rows, err := h.svc.ListGates()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.JSON(okResponse(rows))
}

func (h *Handler) CreateGate(c *fiber.Ctx) error {
	var body struct {
		ZoneID       string `json:"zone_id"`
		Name         string `json:"name"`
		GateType     string `json:"gate_type"`
		Mode         string `json:"mode"`
		LocationDesc string `json:"location_desc"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}
	zoneID, err := uuid.Parse(strings.TrimSpace(body.ZoneID))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "zone_id must be a valid UUID"))
	}

	out, err := h.svc.CreateGate(zoneID, body.Name, body.GateType, body.Mode, body.LocationDesc)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.Status(fiber.StatusCreated).JSON(okResponse(out))
}

func (h *Handler) GetGate(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid gate id"))
	}
	out, err := h.svc.GetGate(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	if out == nil {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("GATE_NOT_FOUND", "gate not found"))
	}
	return c.JSON(okResponse(out))
}

func (h *Handler) UpdateGate(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid gate id"))
	}

	var body struct {
		ZoneID       string `json:"zone_id"`
		Name         string `json:"name"`
		GateType     string `json:"gate_type"`
		Mode         string `json:"mode"`
		LocationDesc string `json:"location_desc"`
		IsActive     bool   `json:"is_active"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}
	zoneID, err := uuid.Parse(strings.TrimSpace(body.ZoneID))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "zone_id must be a valid UUID"))
	}

	out, err := h.svc.UpdateGate(id, zoneID, body.Name, body.GateType, body.Mode, body.LocationDesc, body.IsActive)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	if out == nil {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("GATE_NOT_FOUND", "gate not found"))
	}
	return c.JSON(okResponse(out))
}

func (h *Handler) ListGateDevices(c *fiber.Ctx) error {
	gateID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid gate id"))
	}
	rows, err := h.svc.ListGateDevices(gateID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.JSON(okResponse(rows))
}

func (h *Handler) UpdateGateDeviceStatus(c *fiber.Ctx) error {
	deviceID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid device id"))
	}
	var body struct {
		Status       string `json:"status"`
		ErrorMessage string `json:"error_message"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}
	if strings.TrimSpace(body.Status) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "status is required"))
	}

	if err := h.svc.UpdateGateDeviceStatus(deviceID, body.Status, body.ErrorMessage); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.JSON(okResponse(fiber.Map{"id": deviceID, "status": strings.TrimSpace(strings.ToLower(body.Status))}))
}

func (h *Handler) PairRequest(c *fiber.Ctx) error {
	res, err := h.svc.RequestPairingCode()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.JSON(okResponse(fiber.Map{"code": res.Code, "expires_at": res.ExpiresAt, "status": PairingStatusPending}))
}

func (h *Handler) PairConfirm(c *fiber.Ctx) error {
	var body struct {
		Code   string `json:"code"`
		GateID string `json:"gate_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}
	if strings.TrimSpace(body.Code) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "code is required"))
	}
	gateID, err := uuid.Parse(strings.TrimSpace(body.GateID))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "gate_id must be a valid UUID"))
	}

	if err := h.svc.ConfirmPairingCode(body.Code, gateID, userIDFromCtx(c)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	return c.JSON(okResponse(fiber.Map{"code": strings.ToUpper(strings.TrimSpace(body.Code)), "status": PairingStatusConfirmed}))
}

func (h *Handler) PairVerify(c *fiber.Ctx) error {
	code := strings.TrimSpace(c.Params("code"))
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "code is required"))
	}
	res, err := h.svc.VerifyPairingCode(code)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	data := fiber.Map{
		"code":       strings.ToUpper(code),
		"status":     res.Status,
		"expires_at": res.ExpiresAt,
	}
	if res.GateID != nil {
		data["gate_id"] = *res.GateID
	}
	if res.GateToken != "" {
		data["gate_token"] = res.GateToken
	}
	return c.JSON(okResponse(data))
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
	return fiber.Map{
		"success": false, "data": nil, "meta": fiber.Map{},
		"error": fiber.Map{"code": code, "message": message},
	}
}
