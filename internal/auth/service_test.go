// internal/auth/service_test.go
package auth_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/yyypluto/parkieee-api/internal/auth"
	authmocks "github.com/yyypluto/parkieee-api/internal/auth/mocks"
	usermodule "github.com/yyypluto/parkieee-api/internal/user"
)

func makeUser(passwordPlain string) *auth.User {
	svc := auth.NewService(nil, "secret", 15)
	hash, _ := svc.HashPassword(passwordPlain)
	return &auth.User{
		ID:           uuid.New(),
		Username:     "admin",
		PasswordHash: hash,
		IsActive:     true,
		RoleID:       uuid.New(),
		Role:         usermodule.Role{Name: "admin"},
	}
}

func TestService_Login_Success(t *testing.T) {
	repo := authmocks.NewRepo(t)
	svc := auth.NewService(repo, "secret", 15)

	user := makeUser("password123")
	repo.On("FindUserByUsername", "admin").Return(user, nil)
	repo.On("FindPermissionsByRoleID", user.RoleID).Return([]string{"transaction:write"}, nil)
	repo.On("CreateSession", mock.Anything).Return(nil)
	repo.On("CreateRefreshToken", mock.Anything).Return(nil)
	repo.On("WriteLoginLog", mock.Anything).Return(nil)
	repo.On("UpsertLoginStats", user.ID, true, "127.0.0.1").Return(nil)

	result, err := svc.Login("admin", "password123", "127.0.0.1", "test-agent")
	assert.NoError(t, err)
	assert.NotEmpty(t, result.AccessToken)
	assert.NotEmpty(t, result.RefreshToken)
	assert.NotEmpty(t, result.CsrfToken)
	assert.Equal(t, user.ID, result.UserID)
	assert.Equal(t, user.Role.Name, result.Role)
	assert.Equal(t, []string{"transaction:write"}, result.Permissions)
}

func TestService_Login_WrongPassword(t *testing.T) {
	repo := authmocks.NewRepo(t)
	svc := auth.NewService(repo, "secret", 15)

	user := makeUser("correct")
	repo.On("FindUserByUsername", "admin").Return(user, nil)
	repo.On("WriteLoginLog", mock.Anything).Return(nil)
	repo.On("UpsertLoginStats", user.ID, false, "127.0.0.1").Return(nil)

	_, err := svc.Login("admin", "wrong", "127.0.0.1", "agent")
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestService_Login_UserNotFound(t *testing.T) {
	repo := authmocks.NewRepo(t)
	svc := auth.NewService(repo, "secret", 15)
	repo.On("FindUserByUsername", "nobody").Return(nil, nil)

	_, err := svc.Login("nobody", "x", "", "")
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestService_GetCurrentUser_Success(t *testing.T) {
	repo := authmocks.NewRepo(t)
	svc := auth.NewService(repo, "secret", 15)

	user := makeUser("password123")
	repo.On("FindUserByID", user.ID).Return(user, nil)
	repo.On("FindPermissionsByRoleID", user.RoleID).Return([]string{"transaction:read"}, nil)

	result, err := svc.GetCurrentUser(user.ID.String(), "session-1")
	assert.NoError(t, err)
	assert.Equal(t, user.ID, result.ID)
	assert.Equal(t, user.Role.Name, result.Role)
	assert.Equal(t, []string{"transaction:read"}, result.Permissions)
	assert.Equal(t, "session-1", result.SessionID)
}

func TestService_GetCurrentUser_InvalidID(t *testing.T) {
	repo := authmocks.NewRepo(t)
	svc := auth.NewService(repo, "secret", 15)

	_, err := svc.GetCurrentUser("not-a-uuid", "session-1")
	assert.ErrorIs(t, err, auth.ErrTokenInvalid)
}
