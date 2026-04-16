package user_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/yyypluto/parkieee-api/internal/user"
	usermocks "github.com/yyypluto/parkieee-api/internal/user/mocks"
	"golang.org/x/crypto/bcrypt"
)

func TestCreateUser_DuplicateUsername_ReturnsError(t *testing.T) {
	repo := usermocks.NewRepo(t)
	svc := user.NewService(repo)
	svc.SetPasswordHasher(func(plain string) (string, error) {
		return string([]byte("dummy")), nil
	})

	repo.On("FindUserByUsername", "existing-user").Return(&user.User{ID: uuid.New(), Username: "existing-user"}, nil)

	_, err := svc.CreateUser(user.CreateInput{
		Name:     "Existing User",
		Username: "existing-user",
		Email:    "existing@example.com",
		Password: "password123",
		RoleID:   uuid.New(),
		ActorID:  uuid.New(),
	})

	assert.ErrorIs(t, err, user.ErrDuplicateUsername)
	repo.AssertExpectations(t)
}

func TestDeactivateUser_SelfAction_ReturnsError(t *testing.T) {
	repo := usermocks.NewRepo(t)
	svc := user.NewService(repo)
	svc.SetPasswordHasher(func(plain string) (string, error) {
		return string([]byte("dummy")), nil
	})

	id := uuid.New()
	err := svc.DeactivateUser(id, id)

	assert.ErrorIs(t, err, user.ErrSelfActionForbidden)
	repo.AssertNotCalled(t, "SetUserActive", mock.Anything, mock.Anything)
}

func TestResetPassword_HashesNewPassword(t *testing.T) {
	repo := usermocks.NewRepo(t)
	svc := user.NewService(repo)
	svc.SetPasswordHasher(func(plain string) (string, error) {
		hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
		return string(hash), err
	})

	userID := uuid.New()
	actorID := uuid.New()

	repo.On("FindUserByID", userID).Return(&user.User{ID: userID, Username: "target-user"}, nil)
	repo.On("UpdatePassword", userID, mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		hash := args.String(1)
		assert.NotEqual(t, "new-password-123", hash)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte("new-password-123")))
	}).Return(nil)

	err := svc.ResetPassword(userID, actorID, "new-password-123")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
