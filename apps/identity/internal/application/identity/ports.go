package identity

import (
	"context"
	"time"

	domain "github.com/chouaib-skitou/streamweave/apps/identity/internal/domain/identity"
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
	FindSessionByID(context.Context, uuid.UUID) (domain.Session, error)
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
	LockActiveSession(context.Context, uuid.UUID) (bool, error)
	RotateSession(context.Context, uuid.UUID, uuid.UUID, time.Time) (int64, error)
	RevokeSessionFamily(context.Context, uuid.UUID, time.Time, string) error
	RevokeSession(context.Context, uuid.UUID, time.Time, string) error
	RevokeUserSessions(context.Context, uuid.UUID, time.Time, string) error
	UpdateUserStatus(context.Context, uuid.UUID, domain.UserStatus, time.Time) (int64, error)
	UpdatePassword(context.Context, uuid.UUID, string, time.Time) error
	AssignRole(context.Context, uuid.UUID, string, time.Time, uuid.NullUUID) (int64, error)
	RemoveRole(context.Context, uuid.UUID, string) (int64, error)
	HasActiveRole(context.Context, uuid.UUID, string) (bool, error)
	CountActiveAdministrators(context.Context) (int64, error)
	CreateResetToken(context.Context, uuid.UUID, uuid.UUID, []byte, time.Time, time.Time) error
	ConsumeResetToken(context.Context, uuid.UUID, time.Time) (int64, error)
	CreateVerificationToken(context.Context, uuid.UUID, uuid.UUID, []byte, time.Time, time.Time) error
	ConsumeVerificationToken(context.Context, uuid.UUID, time.Time) (int64, error)
	MarkEmailVerified(context.Context, uuid.UUID, time.Time) (int64, error)
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

// CollectionStore contains bounded collection queries. It is kept separate
// from Store so existing adapters and tests can migrate without changing the
// authentication contract.
type CollectionStore interface {
	ListUsersPage(context.Context, UserPageQuery) (UserPage, error)
	ListSessionsPage(context.Context, uuid.UUID, SessionPageQuery) (SessionPage, error)
}

type UserPageQuery struct {
	Status          string
	Role            string
	EmailPrefix     string
	CursorCreatedAt *time.Time
	CursorID        uuid.UUID
	Limit           int32
}

type UserPageItem struct {
	User  domain.User
	Roles []string
}

type UserPage struct {
	Items       []UserPageItem
	HasMore     bool
	LastCreated time.Time
	LastID      uuid.UUID
}

type SessionPageQuery struct {
	CursorCreatedAt *time.Time
	CursorID        uuid.UUID
	Limit           int32
}

type SessionPage struct {
	Items       []domain.Session
	HasMore     bool
	LastCreated time.Time
	LastID      uuid.UUID
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

type DimensionLimiter interface {
	AllowMany(context.Context, []string) (bool, error)
}

type Revocations interface {
	Revoke(context.Context, string, time.Duration) error
}

type Mailer interface {
	Send(context.Context, string, string, string) error
}

// Mailbox exposes local-only delivery inspection for manual development tests.
// Production mailers must not implement this port.
type Mailbox interface {
	Latest(context.Context, string, string) (string, error)
}
