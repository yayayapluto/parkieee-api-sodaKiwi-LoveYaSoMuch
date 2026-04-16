package user

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrDuplicateUsername = errors.New("duplicate username")
var ErrSelfActionForbidden = errors.New("self action forbidden")
var ErrImmutableField = errors.New("immutable field")
var ErrNotFound = errors.New("not found")
var ErrRoleNotFound = errors.New("role not found")
var ErrValidation = errors.New("validation error")

type Repo interface {
	ListUsers(filter ListFilter) ([]User, int64, error)
	FindUserByID(id uuid.UUID) (*User, error)
	FindUserByUsername(username string) (*User, error)
	CreateUser(u *User) error
	UpdateUser(u *User) error
	SetUserActive(id uuid.UUID, isActive bool) error
	UpdatePassword(id uuid.UUID, passwordHash string) error
	RoleExists(roleID uuid.UUID) (bool, error)
}

type ListFilter struct {
	Page     int
	Limit    int
	RoleName string
	IsActive *bool
	Search   string
}

type CreateInput struct {
	Name     string
	Username string
	Email    string
	Password string
	RoleID   uuid.UUID
	ActorID  uuid.UUID
}

type UpdateInput struct {
	Name     string
	Email    string
	RoleID   uuid.UUID
	Username *string
	ActorID  uuid.UUID
}

type Service struct {
	repo         Repo
	hashPassword func(string) (string, error)
}

func NewService(repo Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) SetPasswordHasher(hasher func(string) (string, error)) {
	s.hashPassword = hasher
}

func (s *Service) ListUsers(filter ListFilter) ([]User, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	return s.repo.ListUsers(filter)
}

func (s *Service) CreateUser(in CreateInput) (*User, error) {
	if in.Name == "" || in.Username == "" || in.Email == "" || in.Password == "" || in.RoleID == uuid.Nil {
		return nil, ErrValidation
	}

	if s.hashPassword == nil {
		return nil, errors.New("password hasher is not configured")
	}

	existing, err := s.repo.FindUserByUsername(in.Username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateUsername
	}

	roleExists, err := s.repo.RoleExists(in.RoleID)
	if err != nil {
		return nil, err
	}
	if !roleExists {
		return nil, ErrRoleNotFound
	}

	hash, err := s.hashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	u := &User{
		ID:           uuid.New(),
		Name:         in.Name,
		Username:     in.Username,
		Email:        in.Email,
		PasswordHash: hash,
		RoleID:       in.RoleID,
		IsActive:     true,
		CreatedBy:    &in.ActorID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.CreateUser(u); err != nil {
		return nil, err
	}

	created, err := s.repo.FindUserByID(u.ID)
	if err != nil {
		return nil, err
	}
	if created == nil {
		return u, nil
	}
	return created, nil
}

func (s *Service) GetUserByID(id uuid.UUID) (*User, error) {
	u, err := s.repo.FindUserByID(id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrNotFound
	}
	return u, nil
}

func (s *Service) UpdateUser(id uuid.UUID, in UpdateInput) (*User, error) {
	u, err := s.repo.FindUserByID(id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrNotFound
	}

	if in.Username != nil && *in.Username != u.Username {
		return nil, ErrImmutableField
	}

	if id == in.ActorID && in.RoleID != uuid.Nil && in.RoleID != u.RoleID {
		return nil, ErrSelfActionForbidden
	}

	if in.RoleID != uuid.Nil {
		roleExists, err := s.repo.RoleExists(in.RoleID)
		if err != nil {
			return nil, err
		}
		if !roleExists {
			return nil, ErrRoleNotFound
		}
		u.RoleID = in.RoleID
	}

	if in.Name != "" {
		u.Name = in.Name
	}
	if in.Email != "" {
		u.Email = in.Email
	}

	if err := s.repo.UpdateUser(u); err != nil {
		return nil, err
	}

	updated, err := s.repo.FindUserByID(id)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrNotFound
	}
	return updated, nil
}

func (s *Service) DeactivateUser(id, actorID uuid.UUID) error {
	if id == actorID {
		return ErrSelfActionForbidden
	}
	u, err := s.repo.FindUserByID(id)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrNotFound
	}
	return s.repo.SetUserActive(id, false)
}

func (s *Service) ResetPassword(id, actorID uuid.UUID, newPassword string) error {
	if id == actorID {
		return ErrSelfActionForbidden
	}
	if newPassword == "" {
		return ErrValidation
	}

	if s.hashPassword == nil {
		return errors.New("password hasher is not configured")
	}

	u, err := s.repo.FindUserByID(id)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrNotFound
	}

	hash, err := s.hashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(id, hash)
}
