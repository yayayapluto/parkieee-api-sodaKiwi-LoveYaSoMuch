package vehicle

import (
	"math"
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

func (h *Handler) ListVehicleTypes(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	rows, total, err := h.svc.ListVehicleTypes(VehicleTypeListFilter{
		Page:   page,
		Limit:  limit,
		Search: c.Query("search"),
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	page, limit = normalizePage(page, limit)
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if total == 0 {
		totalPages = 0
	}

	data := make([]fiber.Map, 0, len(rows))
	for _, row := range rows {
		data = append(data, toVehicleTypeResponse(row))
	}

	return c.JSON(okResponseWithMeta(data, fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
	}))
}

func (h *Handler) CreateVehicleType(c *fiber.Ctx) error {
	var body struct {
		Name        string `json:"name"`
		MinimumFee  int    `json:"minimum_fee"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}

	created, err := h.svc.CreateVehicleType(VehicleTypeInput{
		Name:        body.Name,
		MinimumFee:  body.MinimumFee,
		Description: body.Description,
	})
	if err == ErrValidation {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "name is required and minimum_fee must be >= 0"))
	}
	if err == ErrDuplicate {
		return c.Status(fiber.StatusConflict).JSON(errResponse("ERR_DUPLICATE_VEHICLE_TYPE", "vehicle type name already exists"))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(okResponse(toVehicleTypeResponse(*created)))
}

func (h *Handler) UpdateVehicleType(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid vehicle type id"))
	}

	var body struct {
		Name        string `json:"name"`
		MinimumFee  int    `json:"minimum_fee"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}

	updated, err := h.svc.UpdateVehicleType(id, VehicleTypeInput{
		Name:        body.Name,
		MinimumFee:  body.MinimumFee,
		Description: body.Description,
	})
	if err == ErrValidation {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "name is required and minimum_fee must be >= 0"))
	}
	if err == ErrDuplicate {
		return c.Status(fiber.StatusConflict).JSON(errResponse("ERR_DUPLICATE_VEHICLE_TYPE", "vehicle type name already exists"))
	}
	if err == ErrNotFound {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("ERR_NOT_FOUND", "vehicle type not found"))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.JSON(okResponse(toVehicleTypeResponse(*updated)))
}

func (h *Handler) DeleteVehicleType(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid vehicle type id"))
	}

	err = h.svc.DeleteVehicleType(id)
	if err == ErrValidation {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid vehicle type id"))
	}
	if err == ErrNotFound {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("ERR_NOT_FOUND", "vehicle type not found"))
	}
	if err == ErrInUse {
		return c.Status(fiber.StatusConflict).JSON(errResponse("ERR_IN_USE", "vehicle type is used by existing vehicle"))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.JSON(okResponse(fiber.Map{"message": "vehicle type deleted"}))
}

func (h *Handler) ListVehicles(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	var vehicleTypeID *uuid.UUID
	if raw := c.Query("vehicle_type_id"); raw != "" {
		parsedID, err := uuid.Parse(raw)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "vehicle_type_id must be a valid UUID"))
		}
		vehicleTypeID = &parsedID
	}

	rows, total, err := h.svc.ListVehicles(VehicleListFilter{
		Page:          page,
		Limit:         limit,
		PlateNumber:   c.Query("plate_number"),
		VehicleTypeID: vehicleTypeID,
		Search:        c.Query("search"),
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	page, limit = normalizePage(page, limit)
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if total == 0 {
		totalPages = 0
	}

	data := make([]fiber.Map, 0, len(rows))
	for _, row := range rows {
		data = append(data, toVehicleResponse(row))
	}

	return c.JSON(okResponseWithMeta(data, fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
	}))
}

func (h *Handler) CreateVehicle(c *fiber.Ctx) error {
	var body struct {
		PlateNumber   string `json:"plate_number"`
		VehicleTypeID string `json:"vehicle_type_id"`
		Source        string `json:"source"`
		Notes         string `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}

	vehicleTypeID, err := uuid.Parse(body.VehicleTypeID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "vehicle_type_id must be a valid UUID"))
	}

	created, err := h.svc.CreateVehicle(VehicleInput{
		PlateNumber:   body.PlateNumber,
		VehicleTypeID: vehicleTypeID,
		Source:        body.Source,
		Notes:         body.Notes,
	})
	if err == ErrValidation {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "plate_number and vehicle_type_id are required"))
	}
	if err == ErrNotFound {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("ERR_VEHICLE_TYPE_NOT_FOUND", "vehicle type not found"))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(okResponse(toVehicleResponse(*created)))
}

func (h *Handler) GetVehicle(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid vehicle id"))
	}

	v, err := h.svc.GetVehicleByID(id)
	if err == ErrValidation {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid vehicle id"))
	}
	if err == ErrNotFound {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("ERR_NOT_FOUND", "vehicle not found"))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.JSON(okResponse(toVehicleResponse(*v)))
}

func (h *Handler) UpdateVehicle(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid vehicle id"))
	}

	var body struct {
		VehicleTypeID *string `json:"vehicle_type_id"`
		Notes         *string `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}

	var vehicleTypeID *uuid.UUID
	if body.VehicleTypeID != nil {
		parsedID, err := uuid.Parse(*body.VehicleTypeID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "vehicle_type_id must be a valid UUID"))
		}
		vehicleTypeID = &parsedID
	}

	updated, err := h.svc.UpdateVehicle(id, VehicleUpdateInput{
		VehicleTypeID: vehicleTypeID,
		Notes:         body.Notes,
	})
	if err == ErrValidation {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request data"))
	}
	if err == ErrNotFound {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("ERR_NOT_FOUND", "vehicle or vehicle type not found"))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.JSON(okResponse(toVehicleResponse(*updated)))
}

func (h *Handler) ListRFIDCards(c *fiber.Ctx) error {
	vehicleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid vehicle id"))
	}

	var isActive *bool
	if raw := c.Query("is_active"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "is_active must be true or false"))
		}
		isActive = &parsed
	}

	rows, err := h.svc.ListRFIDCards(vehicleID, isActive)
	if err == ErrValidation {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid vehicle id"))
	}
	if err == ErrNotFound {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("ERR_NOT_FOUND", "vehicle not found"))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	data := make([]fiber.Map, 0, len(rows))
	for _, row := range rows {
		data = append(data, toRFIDResponse(row))
	}

	return c.JSON(okResponse(data))
}

func (h *Handler) AssignRFID(c *fiber.Ctx) error {
	vehicleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid vehicle id"))
	}

	var body struct {
		CardUID string `json:"card_uid"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}

	card, err := h.svc.AssignRFID(vehicleID, body.CardUID)
	if err == ErrValidation {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "vehicle id and card_uid are required"))
	}
	if err == ErrNotFound {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("ERR_NOT_FOUND", "vehicle not found"))
	}
	if err == ErrCardAlreadyAssigned {
		return c.Status(fiber.StatusConflict).JSON(errResponse("ERR_CARD_ALREADY_ASSIGNED", "card uid is already assigned"))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(okResponse(toRFIDResponse(*card)))
}

func (h *Handler) DeactivateRFID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid rfid card id"))
	}

	actorID, err := actorIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResponse("TOKEN_EXPIRED", "invalid token claims"))
	}

	err = h.svc.DeactivateRFID(id, actorID)
	if err == ErrValidation {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request data"))
	}
	if err == ErrNotFound {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("ERR_NOT_FOUND", "rfid card not found"))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.JSON(okResponse(fiber.Map{"message": "rfid card deactivated"}))
}

func actorIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	raw, ok := c.Locals("user_id").(string)
	if !ok || raw == "" {
		return uuid.Nil, fiber.ErrUnauthorized
	}
	return uuid.Parse(raw)
}

func toVehicleTypeResponse(vt VehicleType) fiber.Map {
	return fiber.Map{
		"id":          vt.ID,
		"name":        vt.Name,
		"minimum_fee": vt.MinimumFee,
		"description": vt.Description,
		"created_at":  vt.CreatedAt,
	}
}

func toVehicleResponse(v Vehicle) fiber.Map {
	var vehicleType any = nil
	if v.VehicleType != nil {
		vehicleType = toVehicleTypeResponse(*v.VehicleType)
	}

	return fiber.Map{
		"id":              v.ID,
		"plate_number":    v.PlateNumber,
		"vehicle_type_id": v.VehicleTypeID,
		"source":          v.Source,
		"notes":           v.Notes,
		"tenant_id":       v.TenantID,
		"created_at":      v.CreatedAt,
		"updated_at":      v.UpdatedAt,
		"vehicle_type":    vehicleType,
	}
}

func toRFIDResponse(c RFIDCard) fiber.Map {
	return fiber.Map{
		"id":             c.ID,
		"card_uid":       c.CardUID,
		"vehicle_id":     c.VehicleID,
		"is_active":      c.IsActive,
		"created_at":     c.CreatedAt,
		"deactivated_at": c.DeactivatedAt,
		"deactivated_by": c.DeactivatedBy,
	}
}

func okResponse(data any) fiber.Map {
	return fiber.Map{"success": true, "data": data, "meta": fiber.Map{}, "error": nil}
}

func okResponseWithMeta(data any, meta any) fiber.Map {
	return fiber.Map{"success": true, "data": data, "meta": meta, "error": nil}
}

func errResponse(code, message string) fiber.Map {
	return fiber.Map{
		"success": false, "data": nil, "meta": fiber.Map{},
		"error": fiber.Map{"code": code, "message": message},
	}
}
