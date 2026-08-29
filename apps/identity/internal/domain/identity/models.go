package identity

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type UserStatus string

const (
	UserActive              UserStatus = "ACTIVE"
	UserDisabled            UserStatus = "DISABLED"
	UserPendingVerification UserStatus = "PENDING_VERIFICATION"
)

type User struct {
	ID              uuid.UUID
	Email           string
	EmailNormalized string
	PasswordHash    string
	Status          UserStatus
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

type SessionStatus string

const (
	SessionActive  SessionStatus = "ACTIVE"
	SessionRotated SessionStatus = "ROTATED"
	SessionRevoked SessionStatus = "REVOKED"
)

type Session struct {
	ID         uuid.UUID
	FamilyID   uuid.UUID
	UserID     uuid.UUID
	Status     SessionStatus
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

func (s Session) IsUsable(now time.Time) bool {
	return s.Status == SessionActive && now.Before(s.ExpiresAt)
}

type Credentials struct {
	AccessToken       string
	RefreshToken      string
	AccessTokenExpiry time.Duration
	RefreshExpiry     time.Duration
}

type RolePermission struct {
	Role  string
	Scope string
}

type ServicePrincipal struct {
	ID       uuid.UUID
	ClientID string
	Status   string
	Scopes   []string
	Audience string
}

type ServiceCredential struct {
	ID                 uuid.UUID
	ServicePrincipalID uuid.UUID
	SecretHash         string
	Status             string
	ValidFrom          time.Time
	RetiredAt          *time.Time
}

type OneTimeToken struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	ExpiresAt  time.Time
	ConsumedAt *time.Time
}

func (t OneTimeToken) IsUsable(now time.Time) bool {
	return t.ConsumedAt == nil && now.Before(t.ExpiresAt)
}
