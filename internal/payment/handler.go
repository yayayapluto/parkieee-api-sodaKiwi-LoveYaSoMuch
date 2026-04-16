package payment

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/yyypluto/parkieee-api/pkg/midtrans"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Cash(c *fiber.Ctx) error {
	var body struct {
		TransactionID string `json:"transaction_id"`
		CashTendered  int    `json:"cash_tendered"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}

	txID, err := uuid.Parse(strings.TrimSpace(body.TransactionID))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "transaction_id must be a valid UUID"))
	}

	result, err := h.svc.PayCash(c.UserContext(), CashInput{
		TransactionID: txID,
		CashTendered:  body.CashTendered,
		ActorID:       userIDFromCtx(c),
	})
	if err != nil {
		return handleAppErr(c, err)
	}

	return c.JSON(okResponse(paymentToMap(result)))
}

func (h *Handler) QRIS(c *fiber.Ctx) error {
	var body struct {
		TransactionID string `json:"transaction_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}

	txID, err := uuid.Parse(strings.TrimSpace(body.TransactionID))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "transaction_id must be a valid UUID"))
	}

	result, err := h.svc.CreateQRIS(c.UserContext(), QRISInput{
		TransactionID: txID,
		ActorID:       userIDFromCtx(c),
	})
	if err != nil {
		return handleAppErr(c, err)
	}

	return c.JSON(okResponse(paymentToMap(result)))
}

func (h *Handler) MidtransCallback(c *fiber.Ctx) error {
	payload, err := midtrans.ParseCallback(strings.NewReader(string(c.Body())))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid callback payload"))
	}

	if err := h.svc.HandleMidtransCallback(c.UserContext(), payload, c.Body()); err != nil {
		// Still return 200 for Midtrans, but include error for observability.
		return c.Status(fiber.StatusOK).JSON(errResponse("CALLBACK_ERROR", err.Error()))
	}

	return c.Status(fiber.StatusOK).JSON(okResponse(fiber.Map{"processed": true}))
}

func (h *Handler) GetByTransaction(c *fiber.Ctx) error {
	txID, err := uuid.Parse(c.Params("transaction_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "transaction_id must be a valid UUID"))
	}

	payment, err := h.svc.GetByTransactionID(c.UserContext(), txID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}
	if payment == nil {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("PAYMENT_NOT_FOUND", "payment not found"))
	}

	return c.JSON(okResponse(paymentToMap(payment)))
}

func (h *Handler) Refund(c *fiber.Ctx) error {
	var body struct {
		TransactionID string `json:"transaction_id"`
		Amount        int    `json:"amount"`
		Reason        string `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}

	txID, err := uuid.Parse(strings.TrimSpace(body.TransactionID))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "transaction_id must be a valid UUID"))
	}

	ref, err := h.svc.CreateRefund(c.UserContext(), RefundInput{
		TransactionID: txID,
		Amount:        body.Amount,
		Reason:        body.Reason,
		ActorID:       userIDFromCtx(c),
	})
	if err != nil {
		return handleAppErr(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(okResponse(refundToMap(ref)))
}

func (h *Handler) ListRefunds(c *fiber.Ctx) error {
	txID, err := uuid.Parse(c.Params("transaction_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "transaction_id must be a valid UUID"))
	}

	rows, err := h.svc.ListRefunds(c.UserContext(), txID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	out := make([]fiber.Map, 0, len(rows))
	for i := range rows {
		out = append(out, refundToMap(&rows[i]))
	}
	return c.JSON(okResponse(out))
}

func handleAppErr(c *fiber.Ctx, err error) error {
	appErr, ok := err.(*AppError)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	status := fiber.StatusBadRequest
	switch appErr.Code {
	case "TRANSACTION_NOT_FOUND":
		status = fiber.StatusNotFound
	case "PAYMENT_NOT_FOUND":
		status = fiber.StatusNotFound
	case "INVALID_TRANSACTION_STATUS", "PAYMENT_EXISTS":
		status = fiber.StatusConflict
	case "INSUFFICIENT_CASH":
		status = fiber.StatusBadRequest
	case "PAYMENT_GATEWAY_ERROR":
		status = fiber.StatusBadGateway
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

func paymentToMap(p *Payment) fiber.Map {
	if p == nil {
		return fiber.Map{}
	}
	out := fiber.Map{
		"id":                p.ID,
		"transaction_id":    p.TransactionID,
		"method":            p.Method,
		"amount":            p.Amount,
		"status":            p.Status,
		"cash_tendered":     p.CashTendered,
		"cash_change":       p.CashChange,
		"midtrans_order_id": p.MidtransOrderID,
		"qris_string":       p.QRISString,
		"qris_image_url":    p.QRISImageURL,
		"midtrans_status":   p.MidtransStatus,
		"created_at":        p.CreatedAt,
		"updated_at":        p.UpdatedAt,
	}
	if p.PaidAt != nil {
		out["paid_at"] = *p.PaidAt
	}
	if p.QRISExpiresAt != nil {
		out["qris_expires_at"] = *p.QRISExpiresAt
	}
	if p.HandledByUserID != nil {
		out["handled_by_user_id"] = *p.HandledByUserID
	}
	return out
}

func refundToMap(r *Refund) fiber.Map {
	if r == nil {
		return fiber.Map{}
	}
	out := fiber.Map{
		"id":             r.ID,
		"transaction_id": r.TransactionID,
		"payment_id":     r.PaymentID,
		"amount":         r.Amount,
		"reason":         r.Reason,
		"status":         r.Status,
		"created_at":     r.CreatedAt,
		"updated_at":     r.UpdatedAt,
	}
	if r.RequestedBy != nil {
		out["requested_by"] = *r.RequestedBy
	}
	return out
}

func okResponse(data any) fiber.Map {
	return fiber.Map{"success": true, "data": data, "meta": fiber.Map{}, "error": nil}
}

func errResponse(code, message string) fiber.Map {
	return fiber.Map{"success": false, "data": nil, "meta": fiber.Map{}, "error": fiber.Map{"code": code, "message": message}}
}
