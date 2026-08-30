package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/adapters/outbound/postgres/generated"
	appidentity "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/application/identity"
	domain "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/domain/identity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Store struct {
	db *sql.DB
	q  *generated.Queries
}

type OutboxEvent struct {
	ID            uuid.UUID
	EventType     string
	AggregateType string
	AggregateID   string
	Payload       []byte
	Headers       []byte
	CreatedAt     time.Time
}

func (s *Store) PendingOutboxCount(ctx context.Context) (int64, error) {
	count, err := s.q.CountPendingOutbox(ctx)
	return count, mapDatabaseError(err)
}

func (s *Store) PendingOutbox(ctx context.Context, limit int32) ([]OutboxEvent, error) {
	rows, err := s.q.ListPendingOutbox(ctx, limit)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	result := make([]OutboxEvent, 0, len(rows))
	for _, row := range rows {
		result = append(result, OutboxEvent{ID: row.ID, EventType: row.EventType, AggregateType: row.AggregateType, AggregateID: row.AggregateID, Payload: row.Payload, Headers: row.Headers, CreatedAt: row.CreatedAt})
	}
	return result, nil
}

func (s *Store) MarkOutboxPublished(ctx context.Context, id uuid.UUID, at time.Time) error {
	return mapDatabaseError(s.q.MarkOutboxPublished(ctx, generated.MarkOutboxPublishedParams{ID: id, PublishedAt: sql.NullTime{Time: at, Valid: true}}))
}

func (s *Store) MarkOutboxFailed(ctx context.Context, id uuid.UUID, reason string) error {
	return mapDatabaseError(s.q.MarkOutboxFailed(ctx, generated.MarkOutboxFailedParams{ID: id, LastError: sql.NullString{String: reason, Valid: true}}))
}

func NewStore(connection *Connection) *Store {
	return &Store{db: connection.DB(), q: generated.New(connection.DB())}
}

func (s *Store) FindUserByEmail(ctx context.Context, email string) (domain.User, error) {
	user, err := s.q.GetUserByEmail(ctx, email)
	return mapUser(user), mapDatabaseError(err)
}

func (s *Store) FindUser(ctx context.Context, id uuid.UUID) (domain.User, error) {
	user, err := s.q.GetUserByID(ctx, id)
	return mapUser(user), mapDatabaseError(err)
}

func (s *Store) UserRoles(ctx context.Context, id uuid.UUID) ([]domain.RolePermission, error) {
	rows, err := s.q.ListUserRoles(ctx, id)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	permissions := make([]domain.RolePermission, 0, len(rows))
	for _, row := range rows {
		permissions = append(permissions, domain.RolePermission{Role: row.RoleName, Scope: row.PermissionName})
	}
	return permissions, nil
}

func (s *Store) FindRole(ctx context.Context, name string) error {
	_, err := s.q.GetRoleByName(ctx, name)
	return mapDatabaseError(err)
}

func (s *Store) FindSessionByRefreshHash(ctx context.Context, hash []byte) (domain.Session, error) {
	session, err := s.q.GetSessionByRefreshHash(ctx, hash)
	return mapSession(session.ID, session.FamilyID, session.UserID, session.Status, session.ExpiresAt, session.RevokedAt, session.CreatedAt, session.LastUsedAt), mapDatabaseError(err)
}

func (s *Store) FindSessionByID(ctx context.Context, id uuid.UUID) (domain.Session, error) {
	session, err := s.q.GetSessionByID(ctx, id)
	return mapSession(session.ID, session.FamilyID, session.UserID, session.Status, session.ExpiresAt, session.RevokedAt, session.CreatedAt, session.LastUsedAt), mapDatabaseError(err)
}

func (s *Store) ListSessions(ctx context.Context, userID uuid.UUID) ([]domain.Session, error) {
	rows, err := s.q.ListUserSessions(ctx, userID)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	result := make([]domain.Session, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapSession(row.ID, row.FamilyID, row.UserID, row.Status, row.ExpiresAt, row.RevokedAt, row.CreatedAt, row.LastUsedAt))
	}
	return result, nil
}

func (s *Store) ListSessionsPage(ctx context.Context, userID uuid.UUID, query appidentity.SessionPageQuery) (appidentity.SessionPage, error) {
	rows, err := s.q.ListUserSessionsPage(ctx, generated.ListUserSessionsPageParams{
		UserID: userID, CursorCreatedAt: nullableTimeValue(query.CursorCreatedAt), CursorID: nullableUUIDValue(query.CursorID), PageSize: query.Limit + 1,
	})
	if err != nil {
		return appidentity.SessionPage{}, mapDatabaseError(err)
	}
	page := appidentity.SessionPage{Items: make([]domain.Session, 0, len(rows))}
	for _, row := range rows {
		page.Items = append(page.Items, mapSession(row.ID, row.FamilyID, row.UserID, row.Status, row.ExpiresAt, row.RevokedAt, row.CreatedAt, row.LastUsedAt))
	}
	if len(page.Items) > int(query.Limit) {
		page.HasMore = true
		page.Items = page.Items[:query.Limit]
	}
	if len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		page.LastCreated, page.LastID = last.CreatedAt, last.ID
	}
	return page, nil
}

func (s *Store) ListUsersPage(ctx context.Context, query appidentity.UserPageQuery) (appidentity.UserPage, error) {
	rows, err := s.q.ListUsersPage(ctx, generated.ListUsersPageParams{
		Column1: query.Status, Column2: query.Role, Column3: query.EmailPrefix,
		CursorCreatedAt: nullableTimeValue(query.CursorCreatedAt), CursorID: nullableUUIDValue(query.CursorID), PageSize: query.Limit + 1,
	})
	if err != nil {
		return appidentity.UserPage{}, mapDatabaseError(err)
	}
	hasMore := len(rows) > int(query.Limit)
	if hasMore {
		rows = rows[:query.Limit]
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	roleRows, err := s.q.ListUserRoleNames(ctx, ids)
	if err != nil {
		return appidentity.UserPage{}, mapDatabaseError(err)
	}
	roles := make(map[uuid.UUID][]string, len(ids))
	for _, row := range roleRows {
		roles[row.UserID] = append(roles[row.UserID], row.Name)
	}
	page := appidentity.UserPage{HasMore: hasMore, Items: make([]appidentity.UserPageItem, 0, len(rows))}
	for _, row := range rows {
		page.Items = append(page.Items, appidentity.UserPageItem{User: mapUser(row), Roles: roles[row.ID]})
	}
	if len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1].User
		page.LastCreated, page.LastID = last.CreatedAt, last.ID
	}
	return page, nil
}

func (s *Store) FindServicePrincipal(ctx context.Context, clientID string) (domain.ServicePrincipal, error) {
	principal, err := s.q.GetServicePrincipal(ctx, clientID)
	if err != nil {
		return domain.ServicePrincipal{}, mapDatabaseError(err)
	}
	var scopes []string
	if err := json.Unmarshal(principal.Scopes, &scopes); err != nil {
		return domain.ServicePrincipal{}, fmt.Errorf("decode machine scopes: %w", err)
	}
	return domain.ServicePrincipal{ID: principal.ID, ClientID: principal.ClientID, Status: principal.Status, Scopes: scopes, Audience: principal.Audience}, nil
}

func (s *Store) ActiveServiceCredentials(ctx context.Context, principalID uuid.UUID, now time.Time) ([]domain.ServiceCredential, error) {
	rows, err := s.q.ListActiveServiceCredentials(ctx, generated.ListActiveServiceCredentialsParams{ServicePrincipalID: principalID, ValidFrom: now})
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	result := make([]domain.ServiceCredential, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.ServiceCredential{ID: row.ID, ServicePrincipalID: row.ServicePrincipalID, SecretHash: row.SecretHash, Status: row.Status, ValidFrom: row.ValidFrom, RetiredAt: nullableTime(row.RetiredAt)})
	}
	return result, nil
}

func (s *Store) FindResetToken(ctx context.Context, hash []byte) (domain.OneTimeToken, error) {
	token, err := s.q.GetResetToken(ctx, hash)
	return domain.OneTimeToken{ID: token.ID, UserID: token.UserID, ExpiresAt: token.ExpiresAt, ConsumedAt: nullableTime(token.ConsumedAt)}, mapDatabaseError(err)
}

func (s *Store) FindVerificationToken(ctx context.Context, hash []byte) (domain.OneTimeToken, error) {
	token, err := s.q.GetVerificationToken(ctx, hash)
	return domain.OneTimeToken{ID: token.ID, UserID: token.UserID, ExpiresAt: token.ExpiresAt, ConsumedAt: nullableTime(token.ConsumedAt)}, mapDatabaseError(err)
}

func (s *Store) WithTransaction(ctx context.Context, fn func(appidentity.Transaction) error) error {
	ctx, span := otel.Tracer("identity/postgresql").Start(ctx, "postgres transaction")
	defer span.End()
	span.SetAttributes(attribute.String("db.system", "postgresql"))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return mapDatabaseError(err)
	}
	transaction := &storeTransaction{q: generated.New(tx)}
	if err := fn(transaction); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return mapDatabaseError(err)
	}
	return nil
}

type storeTransaction struct{ q *generated.Queries }

func (t *storeTransaction) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	created, err := t.q.CreateUser(ctx, generated.CreateUserParams{ID: user.ID, Email: user.Email, EmailNormalized: user.EmailNormalized, PasswordHash: user.PasswordHash, Status: string(user.Status), CreatedAt: user.CreatedAt})
	return mapUser(created), mapDatabaseError(err)
}

func (t *storeTransaction) CreateSession(ctx context.Context, session domain.Session, refreshHash, userAgentHash, ipPrefixHash []byte) error {
	_, err := t.q.CreateSession(ctx, generated.CreateSessionParams{ID: session.ID, FamilyID: session.FamilyID, UserID: session.UserID, RefreshTokenHash: refreshHash, ExpiresAt: session.ExpiresAt, CreatedAt: session.CreatedAt, UserAgentHash: userAgentHash, IpPrefixHash: ipPrefixHash, RotatedFromSessionID: nullableUUIDPointer(session.RotatedFromSessionID)})
	return mapDatabaseError(err)
}

func (t *storeTransaction) LockActiveSession(ctx context.Context, id uuid.UUID) (bool, error) {
	_, err := t.q.LockActiveSession(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, domain.ErrNotFound
	}
	return err == nil, mapDatabaseError(err)
}

func (t *storeTransaction) RotateSession(ctx context.Context, id, replacementID uuid.UUID, now time.Time) (int64, error) {
	count, err := t.q.RotateSession(ctx, generated.RotateSessionParams{ID: id, ReplacedBySessionID: uuid.NullUUID{UUID: replacementID, Valid: true}, LastUsedAt: sql.NullTime{Time: now, Valid: true}})
	return count, mapDatabaseError(err)
}

func (t *storeTransaction) RevokeSessionFamily(ctx context.Context, familyID uuid.UUID, now time.Time, reason string) error {
	return mapDatabaseError(t.q.RevokeSessionFamily(ctx, generated.RevokeSessionFamilyParams{FamilyID: familyID, RevokedAt: sql.NullTime{Time: now, Valid: true}, RevokedReason: sql.NullString{String: reason, Valid: true}}))
}

func (t *storeTransaction) RevokeSession(ctx context.Context, id uuid.UUID, now time.Time, reason string) error {
	return mapDatabaseError(t.q.RevokeSessionByID(ctx, generated.RevokeSessionByIDParams{ID: id, RevokedAt: sql.NullTime{Time: now, Valid: true}, RevokedReason: sql.NullString{String: reason, Valid: true}}))
}

func (t *storeTransaction) RevokeUserSessions(ctx context.Context, userID uuid.UUID, now time.Time, reason string) error {
	return mapDatabaseError(t.q.RevokeUserSessions(ctx, generated.RevokeUserSessionsParams{UserID: userID, RevokedAt: sql.NullTime{Time: now, Valid: true}, RevokedReason: sql.NullString{String: reason, Valid: true}}))
}

func (t *storeTransaction) UpdateUserStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus, now time.Time) (int64, error) {
	count, err := t.q.UpdateUserStatus(ctx, generated.UpdateUserStatusParams{ID: id, Status: string(status), UpdatedAt: now})
	return count, mapDatabaseError(err)
}

func (t *storeTransaction) UpdatePassword(ctx context.Context, id uuid.UUID, hash string, now time.Time) error {
	return mapDatabaseError(t.q.UpdatePassword(ctx, generated.UpdatePasswordParams{ID: id, PasswordHash: hash, UpdatedAt: now}))
}

func (t *storeTransaction) AssignRole(ctx context.Context, userID uuid.UUID, role string, now time.Time, assignedBy uuid.NullUUID) (int64, error) {
	count, err := t.q.AssignRole(ctx, generated.AssignRoleParams{UserID: userID, Name: role, AssignedAt: now, AssignedBy: assignedBy})
	return count, mapDatabaseError(err)
}

func (t *storeTransaction) RemoveRole(ctx context.Context, userID uuid.UUID, role string) (int64, error) {
	count, err := t.q.RemoveRole(ctx, generated.RemoveRoleParams{UserID: userID, Name: role})
	return count, mapDatabaseError(err)
}

func (t *storeTransaction) HasActiveRole(ctx context.Context, userID uuid.UUID, role string) (bool, error) {
	value, err := t.q.HasActiveRole(ctx, generated.HasActiveRoleParams{UserID: userID, Name: role})
	return value, mapDatabaseError(err)
}

func (t *storeTransaction) CountActiveAdministrators(ctx context.Context) (int64, error) {
	count, err := t.q.CountActiveAdministrators(ctx)
	return count, mapDatabaseError(err)
}

func (t *storeTransaction) CreateResetToken(ctx context.Context, id uuid.UUID, userID uuid.UUID, hash []byte, expiresAt, createdAt time.Time) error {
	return mapDatabaseError(t.q.CreateResetToken(ctx, generated.CreateResetTokenParams{ID: id, UserID: userID, TokenHash: hash, ExpiresAt: expiresAt, CreatedAt: createdAt}))
}

func (t *storeTransaction) ConsumeResetToken(ctx context.Context, id uuid.UUID, now time.Time) (int64, error) {
	count, err := t.q.ConsumeResetToken(ctx, generated.ConsumeResetTokenParams{ID: id, ConsumedAt: sql.NullTime{Time: now, Valid: true}})
	return count, mapDatabaseError(err)
}

func (t *storeTransaction) CreateVerificationToken(ctx context.Context, id uuid.UUID, userID uuid.UUID, hash []byte, expiresAt, createdAt time.Time) error {
	return mapDatabaseError(t.q.CreateVerificationToken(ctx, generated.CreateVerificationTokenParams{ID: id, UserID: userID, TokenHash: hash, ExpiresAt: expiresAt, CreatedAt: createdAt}))
}

func (t *storeTransaction) ConsumeVerificationToken(ctx context.Context, id uuid.UUID, now time.Time) (int64, error) {
	count, err := t.q.ConsumeVerificationToken(ctx, generated.ConsumeVerificationTokenParams{ID: id, ConsumedAt: sql.NullTime{Time: now, Valid: true}})
	return count, mapDatabaseError(err)
}

func (t *storeTransaction) MarkEmailVerified(ctx context.Context, id uuid.UUID, now time.Time) (int64, error) {
	count, err := t.q.MarkEmailVerified(ctx, generated.MarkEmailVerifiedParams{ID: id, EmailVerifiedAt: sql.NullTime{Time: now, Valid: true}})
	return count, mapDatabaseError(err)
}

func (t *storeTransaction) MarkServicePrincipalUsed(ctx context.Context, id uuid.UUID, now time.Time) error {
	return mapDatabaseError(t.q.MarkServicePrincipalUsed(ctx, generated.MarkServicePrincipalUsedParams{ID: id, LastUsedAt: sql.NullTime{Time: now, Valid: true}}))
}

func (t *storeTransaction) RecordAudit(ctx context.Context, record appidentity.AuditRecord) error {
	return mapDatabaseError(t.q.InsertSecurityAudit(ctx, generated.InsertSecurityAuditParams{ID: record.ID, ActorID: nullableString(record.ActorID), ActorType: nullableString(record.ActorType), Action: record.Action, TargetType: nullableString(record.TargetType), TargetID: nullableString(record.TargetID), Metadata: record.Metadata, CorrelationID: record.CorrelationID, OccurredAt: record.OccurredAt}))
}

func (t *storeTransaction) RecordOutbox(ctx context.Context, record appidentity.OutboxRecord) error {
	return mapDatabaseError(t.q.InsertOutboxEvent(ctx, generated.InsertOutboxEventParams{ID: record.ID, EventType: record.EventType, AggregateType: record.AggregateType, AggregateID: record.AggregateID, Payload: record.Payload, Headers: record.Headers, CreatedAt: record.CreatedAt}))
}

func mapUser(user generated.User) domain.User {
	return domain.User{ID: user.ID, Email: user.Email, EmailNormalized: user.EmailNormalized, PasswordHash: user.PasswordHash, Status: domain.UserStatus(user.Status), EmailVerifiedAt: nullableTime(user.EmailVerifiedAt), CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt}
}

func mapSession(id, familyID, userID uuid.UUID, status string, expiresAt time.Time, revokedAt sql.NullTime, createdAt time.Time, lastUsedAt sql.NullTime) domain.Session {
	return domain.Session{ID: id, FamilyID: familyID, UserID: userID, Status: domain.SessionStatus(status), ExpiresAt: expiresAt, RevokedAt: nullableTime(revokedAt), CreatedAt: createdAt, LastUsedAt: nullableTime(lastUsedAt)}
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func nullableTimeValue(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value.UTC(), Valid: true}
}

func nullableUUIDValue(value uuid.UUID) uuid.NullUUID {
	if value == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: value, Valid: true}
}
func nullableUUIDPointer(value *uuid.UUID) uuid.NullUUID {
	if value == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *value, Valid: true}
}
func nullableString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func mapDatabaseError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrConflict
	}
	return fmt.Errorf("%w: %w", domain.ErrDependency, err)
}
