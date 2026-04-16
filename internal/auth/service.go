// internal/auth/service.go
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrTokenExpired = errors.New("token expired")
var ErrTokenInvalid = errors.New("token invalid")
var ErrUserNotFound = errors.New("user not found")

// Repo is the interface the service depends on (allows mock in tests).
type Repo interface {
	FindUserByUsername(username string) (*User, error)
	FindUserByID(id uuid.UUID) (*User, error)
	FindPermissionsByRoleID(roleID uuid.UUID) ([]string, error)
	CreateSession(s *UserSession) error
	CreateRefreshToken(rt *RefreshToken) error
	FindRefreshToken(tokenHash string) (*RefreshToken, error)
	RevokeRefreshToken(id uuid.UUID, replacedBy *uuid.UUID) error
	RevokeSession(sessionID uuid.UUID) error
	RevokeAllUserRefreshTokens(userID uuid.UUID) error
	WriteLoginLog(l *UserLoginLog) error
	UpsertLoginStats(userID uuid.UUID, success bool, ip string) error
}

type Service struct {
	repo          Repo
	jwtSecret     string
	jwtExpMinutes int
}

func NewService(repo Repo, jwtSecret string, jwtExpMinutes int) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret, jwtExpMinutes: jwtExpMinutes}
}

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	CsrfToken    string
	SessionID    uuid.UUID
	ExpiresAt    time.Time
	UserID       uuid.UUID
	Role         string
	Permissions  []string
}

type Claims struct {
	jwt.RegisteredClaims
	UserID      string   `json:"user_id"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	SessionID   string   `json:"session_id"`
}

func (s *Service) Login(username, password, ip, userAgent string) (*LoginResult, error) {
	user, err := s.repo.FindUserByUsername(username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.writeLog(user.ID, ip, userAgent, false, "wrong password")
		return nil, ErrInvalidCredentials
	}

	permissions, err := s.repo.FindPermissionsByRoleID(user.RoleID)
	if err != nil {
		return nil, err
	}

	sessionID := uuid.New()
	sessionToken := generateToken()
	sessionHash := hashToken(sessionToken)

	session := &UserSession{
		ID:        sessionID,
		UserID:    user.ID,
		TokenHash: sessionHash,
		IPAddress: ip,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := s.repo.CreateSession(session); err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(time.Duration(s.jwtExpMinutes) * time.Minute)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:      user.ID.String(),
		Role:        user.Role.Name,
		Permissions: permissions,
		SessionID:   sessionID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("auth: sign JWT: %w", err)
	}

	refreshRaw := generateToken()
	refreshHash := hashToken(refreshRaw)
	rt := &RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		SessionID: sessionID,
		TokenHash: refreshHash,
		IPAddress: ip,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := s.repo.CreateRefreshToken(rt); err != nil {
		return nil, err
	}

	s.writeLog(user.ID, ip, userAgent, true, "")
	_ = s.repo.UpsertLoginStats(user.ID, true, ip)

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshRaw,
		CsrfToken:    generateCSRFToken(),
		SessionID:    sessionID,
		ExpiresAt:    expiresAt,
		UserID:       user.ID,
		Role:         user.Role.Name,
		Permissions:  permissions,
	}, nil
}

type RefreshResult struct {
	AccessToken  string
	RefreshToken string
	CsrfToken    string
	ExpiresAt    time.Time
}

func (s *Service) Refresh(rawRefreshToken, ip, userAgent string) (*RefreshResult, error) {
	hash := hashToken(rawRefreshToken)
	rt, err := s.repo.FindRefreshToken(hash)
	if err != nil {
		return nil, err
	}
	if rt == nil {
		return nil, ErrTokenInvalid
	}

	user, err := s.repo.FindUserByID(rt.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrTokenInvalid
	}

	permissions, err := s.repo.FindPermissionsByRoleID(user.RoleID)
	if err != nil {
		return nil, err
	}

	// Rotate: revoke old token
	newRefreshRaw := generateToken()
	newID := uuid.New()
	if err := s.repo.RevokeRefreshToken(rt.ID, &newID); err != nil {
		return nil, err
	}

	// Mint new access JWT
	expiresAt := time.Now().Add(time.Duration(s.jwtExpMinutes) * time.Minute)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:      user.ID.String(),
		Role:        user.Role.Name,
		Permissions: permissions,
		SessionID:   rt.SessionID.String(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := t.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}

	newRT := &RefreshToken{
		ID:        newID,
		UserID:    user.ID,
		SessionID: rt.SessionID,
		TokenHash: hashToken(newRefreshRaw),
		IPAddress: ip,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := s.repo.CreateRefreshToken(newRT); err != nil {
		return nil, err
	}

	return &RefreshResult{
		AccessToken:  accessToken,
		RefreshToken: newRefreshRaw,
		CsrfToken:    generateCSRFToken(),
		ExpiresAt:    expiresAt,
	}, nil
}

type CurrentUser struct {
	ID          uuid.UUID
	Role        string
	Permissions []string
	SessionID   string
}

func (s *Service) GetCurrentUser(userIDRaw, sessionID string) (*CurrentUser, error) {
	userID, err := uuid.Parse(userIDRaw)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	permissions, err := s.repo.FindPermissionsByRoleID(user.RoleID)
	if err != nil {
		return nil, err
	}

	return &CurrentUser{
		ID:          user.ID,
		Role:        user.Role.Name,
		Permissions: permissions,
		SessionID:   sessionID,
	}, nil
}

func (s *Service) Logout(rawRefreshToken string) error {
	hash := hashToken(rawRefreshToken)
	rt, err := s.repo.FindRefreshToken(hash)
	if err != nil {
		return err
	}
	if rt == nil {
		return nil // already gone
	}
	_ = s.repo.RevokeSession(rt.SessionID)
	return s.repo.RevokeRefreshToken(rt.ID, nil)
}

// ParseAccessToken validates a JWT and returns Claims.
func (s *Service) ParseAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrTokenInvalid
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

func (s *Service) HashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(h), err
}

func (s *Service) writeLog(userID uuid.UUID, ip, userAgent string, success bool, reason string) {
	l := &UserLoginLog{
		ID:            uuid.New(),
		UserID:        userID,
		IPAddress:     ip,
		UserAgent:     userAgent,
		AttemptType:   "password",
		Success:       success,
		FailureReason: reason,
		AttemptedAt:   time.Now(),
	}
	_ = s.repo.WriteLoginLog(l)
	_ = s.repo.UpsertLoginStats(userID, success, ip)
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
