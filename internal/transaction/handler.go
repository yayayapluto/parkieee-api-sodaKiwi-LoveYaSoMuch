package transaction

import (
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Entry(c *fiber.Ctx) error {
	var body struct {
		GateID      string  `json:"gate_id"`
		Method      string  `json:"method"`
		RFIDCardUID *string `json:"rfid_card_uid"`
		PhotoPath   string  `json:"photo_path"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}

	gateID, err := uuid.Parse(strings.TrimSpace(body.GateID))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "gate_id must be a valid UUID"))
	}

	actorID := userIDFromCtx(c)
	txData, err := h.svc.Entry(c.UserContext(), EntryInput{
		GateID:      gateID,
		Method:      strings.ToLower(strings.TrimSpace(body.Method)),
		RFIDCardUID: body.RFIDCardUID,
		PhotoPath:   body.PhotoPath,
		ActorID:     actorID,
	})
	if err != nil {
		return handleAppErr(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(okResponse(txToMap(txData)))
}

func (h *Handler) Exit(c *fiber.Ctx) error {
	var body struct {
		GateID      string  `json:"gate_id"`
		Method      string  `json:"method"`
		RFIDCardUID *string `json:"rfid_card_uid"`
		TicketCode  *string `json:"ticket_code"`
		PhotoPath   string  `json:"photo_path"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}

	gateID, err := uuid.Parse(strings.TrimSpace(body.GateID))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "gate_id must be a valid UUID"))
	}

	actorID := userIDFromCtx(c)
	txData, err := h.svc.Exit(c.UserContext(), ExitInput{
		GateID:      gateID,
		Method:      strings.ToLower(strings.TrimSpace(body.Method)),
		RFIDCardUID: body.RFIDCardUID,
		TicketCode:  body.TicketCode,
		PhotoPath:   body.PhotoPath,
		ActorID:     actorID,
	})
	if err != nil {
		return handleAppErr(c, err)
	}

	return c.JSON(okResponse(txToMap(txData)))
}

func (h *Handler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	filter := ListFilter{
		Page:   page,
		Limit:  limit,
		Status: c.Query("status"),
	}

	if raw := strings.TrimSpace(c.Query("zone_id")); raw != "" {
		zoneID, err := uuid.Parse(raw)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "zone_id must be a valid UUID"))
		}
		filter.ZoneID = &zoneID
	}

	if raw := strings.TrimSpace(c.Query("start_date")); raw != "" {
		start, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "start_date must be RFC3339"))
		}
		filter.StartDate = &start
	}
	if raw := strings.TrimSpace(c.Query("end_date")); raw != "" {
		end, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "end_date must be RFC3339"))
		}
		filter.EndDate = &end
	}

	rows, total, err := h.svc.List(c.UserContext(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if total == 0 {
		totalPages = 0
	}

	data := make([]fiber.Map, 0, len(rows))
	for _, row := range rows {
		data = append(data, txToMap(&row))
	}

	return c.JSON(okResponseWithMeta(data, fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
	}))
}

func (h *Handler) Detail(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid transaction id"))
	}

	out, err := h.svc.Detail(c.UserContext(), id)
	if err != nil {
		return handleAppErr(c, err)
	}

	return c.JSON(okResponse(txToMap(out)))
}

func (h *Handler) Logs(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid transaction id"))
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, total, err := h.svc.ListLogs(c.UserContext(), id, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.JSON(okResponseWithMeta(rows, fiber.Map{
		"page":  page,
		"limit": limit,
		"total": total,
	}))
}

func (h *Handler) UploadEntryPhoto(c *fiber.Ctx) error {
	return h.uploadPhoto(c, "entry")
}

func (h *Handler) UploadExitPhoto(c *fiber.Ctx) error {
	return h.uploadPhoto(c, "exit")
}

func (h *Handler) uploadPhoto(c *fiber.Ctx, photoType string) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "file is required"))
	}
	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "unable to open uploaded file"))
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "unable to read uploaded file"))
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if strings.TrimSpace(contentType) == "" {
		contentType = "image/jpeg"
	}

	url, err := h.svc.UploadPhoto(c.UserContext(), photoType, content, contentType)
	if err != nil {
		return handleAppErr(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(okResponse(fiber.Map{
		"photo_path": url,
		"photo_url":  url,
	}))
}

func handleAppErr(c *fiber.Ctx, err error) error {
	appErr, ok := err.(*AppError)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	status := fiber.StatusBadRequest
	switch appErr.Code {
	case "GATE_NOT_FOUND", "RFID_NOT_FOUND", "TRANSACTION_NOT_FOUND":
		status = fiber.StatusNotFound
	case "DOUBLE_ENTRY":
		status = fiber.StatusConflict
	case "NO_FEE_CONFIG", "INVALID_GATE_TYPE", "INVALID_ENTRY_METHOD":
		status = fiber.StatusBadRequest
	default:
		status = fiber.StatusBadRequest
	}

	return c.Status(status).JSON(errResponse(appErr.Code, appErr.Message))
}

func userIDFromCtx(c *fiber.Ctx) uuid.UUID {
	v, _ := c.Locals("user_id").(string)
	id, err := uuid.Parse(v)
	if err != nil {
		return uuid.Nil
	}
	return id
}

func txToMap(t *Transaction) fiber.Map {
	if t == nil {
		return fiber.Map{}
	}

	data := fiber.Map{
		"id":               t.ID,
		"transaction_code": t.TransactionCode,
		"entry_gate_id":    t.EntryGateID,
		"entry_method":     t.EntryMethod,
		"entry_at":         t.EntryAt,
		"entry_photo_url":  t.EntryPhotoURL,
		"entry_photo_path": t.EntryPhotoPath,
		"status":           t.Status,
		"zone_id":          t.ZoneID,
		"calculated_fee":   t.CalculatedFee,
		"plate_mismatch":   t.PlateMismatch,
		"created_at":       t.CreatedAt,
		"updated_at":       t.UpdatedAt,
	}

	if t.VehicleID != nil {
		data["vehicle_id"] = *t.VehicleID
	}
	if t.RFIDCardID != nil {
		data["rfid_card_id"] = *t.RFIDCardID
	}
	if t.TicketCode != nil {
		data["ticket_code"] = *t.TicketCode
		data["ticket_code_image"] = t.TicketCodeImage
	}
	if t.ExitGateID != nil {
		data["exit_gate_id"] = *t.ExitGateID
	}
	if t.ExitMethod != nil {
		data["exit_method"] = *t.ExitMethod
	}
	if t.ExitAt != nil {
		data["exit_at"] = *t.ExitAt
	}
	if t.FeeConfigID != nil {
		data["fee_config_id"] = *t.FeeConfigID
	}

	return data
}

func okResponse(data any) fiber.Map {
	return fiber.Map{"success": true, "data": data, "meta": fiber.Map{}, "error": nil}
}

func okResponseWithMeta(data, meta any) fiber.Map {
	return fiber.Map{"success": true, "data": data, "meta": meta, "error": nil}
}

func errResponse(code, message string) fiber.Map {
	return fiber.Map{"success": false, "data": nil, "meta": fiber.Map{}, "error": fiber.Map{"code": code, "message": message}}
}
