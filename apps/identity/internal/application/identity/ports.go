package identity

import (
	"context"
	"time"

	domain "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/domain/identity"
	"github.com/google/uuid"
)

type UserStore interface {
	FindUserByEmail(context.Context, string) (domain.User, error)
	FindUser(context.Context, uuid.UUID) (domain.User, error)
	UserRoles(context.Context, uuid.UUID) ([]domain.RolePermission, error)
	FindRole(context.Context, string) error
}

type SessionStore interface {
	FindSessionByRefreshHash(context.Context, []byte) (domain.Session, error)
	ListSessions(context.Context, uuid.UUID) ([]domain.Session, error)
}

type MachineStore interface {
	FindServicePrincipal(context.Context, string) (domain.ServicePrincipal, error)
	ActiveServiceCredentials(context.Context, uuid.UUID, time.Time) ([]domain.ServiceCredential, error)
}

type RecoveryStore interface {
	FindResetToken(context.Context, []byte) (domain.OneTimeToken, error)
	FindVerificationToken(context.Context, []byte) (domain.OneTimeToken, error)
}

type Transaction interface {
	CreateUser(context.Context, domain.User) (domain.User, error)
	CreateSession(context.Context, domain.Session, []byte, []byte, []byte) error
	RotateSession(context.Context, uuid.UUID, uuid.UUID, time.Time) (int64, error)
	RevokeSessionFamily(context.Context, uuid.UUID, time.Time, string) error
	RevokeSession(context.Context, uuid.UUID, time.Time, string) error
	RevokeUserSessions(context.Context, uuid.UUID, time.Time, string) error
	UpdateUserStatus(context.Context, uuid.UUID, domain.UserStatus, time.Time) error
	UpdatePassword(context.Context, uuid.UUID, string, time.Time) error
	AssignRole(context.Context, uuid.UUID, string, time.Time, uuid.NullUUID) error
	RemoveRole(context.Context, uuid.UUID, string) error
	CreateResetToken(context.Context, uuid.UUID, uuid.UUID, []byte, time.Time, time.Time) error
	ConsumeResetToken(context.Context, uuid.UUID, time.Time) (int64, error)
	CreateVerificationToken(context.Context, uuid.UUID, uuid.UUID, []byte, time.Time, time.Time) error
	ConsumeVerificationToken(context.Context, uuid.UUID, time.Time) (int64, error)
	MarkEmailVerified(context.Context, uuid.UUID, time.Time) error
	MarkServicePrincipalUsed(context.Context, uuid.UUID, time.Time) error
	RecordAudit(context.Context, AuditRecord) error
	RecordOutbox(context.Context, OutboxRecord) error
}

type Store interface {
	UserStore
	SessionStore
	MachineStore
	RecoveryStore
	WithTransaction(context.Context, func(Transaction) error) error
}

type AuditRecord struct {
	ID            uuid.UUID
	ActorID       *string
	ActorType     *string
	Action        string
	TargetType    *string
	TargetID      *string
	Metadata      []byte
	CorrelationID string
	OccurredAt    time.Time
}

type OutboxRecord struct {
	ID            uuid.UUID
	EventType     string
	AggregateType string
	AggregateID   string
	Payload       []byte
	Headers       []byte
	CreatedAt     time.Time
}

type Passwords interface {
	Hash(string) (string, error)
	Verify(string, string) bool
}

type Tokens interface {
	NewOpaque(size int) (string, []byte, error)
	HashOpaque(string) []byte
	Sign(subject, audience, kind, sessionID string, roles, scopes []string, ttl time.Duration) (string, error)
}

type Clock interface {
	Now() time.Time
}

type Limiter interface {
	Allow(context.Context, string) (bool, error)
}

type Revocations interface {
	Revoke(context.Context, string, time.Duration) error
}

type Mailer interface {
	Send(context.Context, string, string) error
}
