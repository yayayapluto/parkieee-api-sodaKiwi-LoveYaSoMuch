package user

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

func (h *Handler) ListUsers(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	var isActive *bool
	if raw := c.Query("is_active"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "is_active must be true or false"))
		}
		isActive = &v
	}

	users, total, err := h.svc.ListUsers(ListFilter{
		Page:     page,
		Limit:    limit,
		RoleName: c.Query("role"),
		IsActive: isActive,
		Search:   c.Query("search"),
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	data := make([]fiber.Map, 0, len(users))
	for _, u := range users {
		data = append(data, toUserResponse(u))
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if total == 0 {
		totalPages = 0
	}

	return c.JSON(okResponseWithMeta(data, fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
	}))
}

func (h *Handler) CreateUser(c *fiber.Ctx) error {
	var body struct {
		Name     string `json:"name"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		RoleID   string `json:"role_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}

	roleID, err := uuid.Parse(body.RoleID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "role_id must be a valid UUID"))
	}

	actorID, err := actorIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResponse("TOKEN_EXPIRED", "invalid token claims"))
	}

	u, err := h.svc.CreateUser(CreateInput{
		Name:     body.Name,
		Username: body.Username,
		Email:    body.Email,
		Password: body.Password,
		RoleID:   roleID,
		ActorID:  actorID,
	})
	if err == ErrValidation {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "name, username, email, password, and role_id are required"))
	}
	if err == ErrDuplicateUsername {
		return c.Status(fiber.StatusConflict).JSON(errResponse("ERR_DUPLICATE_USERNAME", "username is already in use"))
	}
	if err == ErrRoleNotFound {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("ERR_ROLE_NOT_FOUND", "role does not exist"))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(okResponse(toUserResponse(*u)))
}

func (h *Handler) GetUser(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid user id"))
	}

	u, err := h.svc.GetUserByID(id)
	if err == ErrNotFound {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("ERR_NOT_FOUND", "user not found"))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.JSON(okResponse(toUserResponse(*u)))
}

func (h *Handler) UpdateUser(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid user id"))
	}

	var body struct {
		Name     string  `json:"name"`
		Email    string  `json:"email"`
		RoleID   string  `json:"role_id"`
		Username *string `json:"username"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}

	var roleID uuid.UUID
	if body.RoleID != "" {
		parsedRoleID, err := uuid.Parse(body.RoleID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "role_id must be a valid UUID"))
		}
		roleID = parsedRoleID
	}

	actorID, err := actorIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResponse("TOKEN_EXPIRED", "invalid token claims"))
	}

	u, err := h.svc.UpdateUser(id, UpdateInput{
		Name:     body.Name,
		Email:    body.Email,
		RoleID:   roleID,
		Username: body.Username,
		ActorID:  actorID,
	})
	if err == ErrImmutableField {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("ERR_IMMUTABLE_FIELD", "username cannot be changed"))
	}
	if err == ErrSelfActionForbidden {
		return c.Status(fiber.StatusForbidden).JSON(errResponse("ERR_SELF_ACTION_FORBIDDEN", "cannot change your own role"))
	}
	if err == ErrRoleNotFound {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("ERR_ROLE_NOT_FOUND", "role does not exist"))
	}
	if err == ErrNotFound {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("ERR_NOT_FOUND", "user not found"))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.JSON(okResponse(toUserResponse(*u)))
}

func (h *Handler) DeactivateUser(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid user id"))
	}

	actorID, err := actorIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResponse("TOKEN_EXPIRED", "invalid token claims"))
	}

	err = h.svc.DeactivateUser(id, actorID)
	if err == ErrSelfActionForbidden {
		return c.Status(fiber.StatusForbidden).JSON(errResponse("ERR_SELF_ACTION_FORBIDDEN", "cannot deactivate your own account"))
	}
	if err == ErrNotFound {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("ERR_NOT_FOUND", "user not found"))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.JSON(okResponse(fiber.Map{"message": "user deactivated"}))
}

func (h *Handler) ResetPassword(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid user id"))
	}

	var body struct {
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "invalid request body"))
	}

	actorID, err := actorIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(errResponse("TOKEN_EXPIRED", "invalid token claims"))
	}

	err = h.svc.ResetPassword(id, actorID, body.NewPassword)
	if err == ErrValidation {
		return c.Status(fiber.StatusBadRequest).JSON(errResponse("VALIDATION_ERROR", "new_password is required"))
	}
	if err == ErrSelfActionForbidden {
		return c.Status(fiber.StatusForbidden).JSON(errResponse("ERR_SELF_ACTION_FORBIDDEN", "cannot reset your own password from this endpoint"))
	}
	if err == ErrNotFound {
		return c.Status(fiber.StatusNotFound).JSON(errResponse("ERR_NOT_FOUND", "user not found"))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.JSON(okResponse(fiber.Map{"message": "password reset"}))
}

func actorIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	raw, ok := c.Locals("user_id").(string)
	if !ok || raw == "" {
		return uuid.Nil, fiber.ErrUnauthorized
	}
	return uuid.Parse(raw)
}

func toUserResponse(u User) fiber.Map {
	return fiber.Map{
		"id":         u.ID,
		"name":       u.Name,
		"username":   u.Username,
		"email":      u.Email,
		"role_id":    u.RoleID,
		"role":       u.Role.Name,
		"is_active":  u.IsActive,
		"created_at": u.CreatedAt,
		"updated_at": u.UpdatedAt,
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
