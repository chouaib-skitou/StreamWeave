package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/mail"
	"sort"
	"strings"
	"time"

	domain "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/domain/identity"
	"github.com/google/uuid"
)

const (
	accessTokenTTL  = 10 * time.Minute
	machineTokenTTL = 5 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
	resetTokenTTL   = 30 * time.Minute
	verifyTokenTTL  = 24 * time.Hour
)

type Service struct {
	store           Store
	passwords       Passwords
	tokens          Tokens
	clock           Clock
	limiter         Limiter
	revocations     Revocations
	mailer          Mailer
	humanAudience   string
	machineAudience string
}

type Options struct {
	Store           Store
	Passwords       Passwords
	Tokens          Tokens
	Clock           Clock
	Limiter         Limiter
	Revocations     Revocations
	Mailer          Mailer
	HumanAudience   string
	MachineAudience string
}

func NewService(options Options) (*Service, error) {
	if options.Store == nil || options.Passwords == nil || options.Tokens == nil || options.Clock == nil {
		return nil, errors.New("identity service dependencies are incomplete")
	}
	if options.HumanAudience == "" || options.MachineAudience == "" {
		return nil, errors.New("identity audiences are required")
	}
	return &Service{store: options.Store, passwords: options.Passwords, tokens: options.Tokens, clock: options.Clock, limiter: options.Limiter, revocations: options.Revocations, mailer: options.Mailer, humanAudience: options.HumanAudience, machineAudience: options.MachineAudience}, nil
}

type RegisterInput struct{ Email, Password string }
type LoginInput struct{ Email, Password, CorrelationID string }
type RefreshInput struct{ RefreshToken, CorrelationID string }
type Credentials struct {
	AccessToken, RefreshToken              string
	AccessTokenExpiresIn, RefreshExpiresIn int
}
type MachineCredentials struct {
	AccessToken string
	ExpiresIn   int
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (domain.User, error) {
	if err := validateCredentials(input.Email, input.Password); err != nil {
		return domain.User{}, err
	}
	if err := s.allow(ctx, "register:"+identityKey(domain.NormalizeEmail(input.Email))); err != nil {
		return domain.User{}, err
	}
	hash, err := s.passwords.Hash(input.Password)
	if err != nil {
		return domain.User{}, err
	}
	verificationToken, verificationHash, err := s.tokens.NewOpaque(32)
	if err != nil {
		return domain.User{}, err
	}
	now := s.clock.Now().UTC()
	user := domain.User{ID: uuid.New(), Email: strings.TrimSpace(input.Email), EmailNormalized: domain.NormalizeEmail(input.Email), PasswordHash: hash, Status: domain.UserActive, CreatedAt: now, UpdatedAt: now}
	err = s.store.WithTransaction(ctx, func(tx Transaction) error {
		created, createErr := tx.CreateUser(ctx, user)
		if createErr != nil {
			return createErr
		}
		user = created
		if err := tx.AssignRole(ctx, user.ID, "customer", now, uuid.NullUUID{}); err != nil {
			return err
		}
		if err := tx.CreateVerificationToken(ctx, uuid.New(), user.ID, verificationHash, now.Add(verifyTokenTTL), now); err != nil {
			return err
		}
		return s.recordAudit(ctx, tx, auditInput{Action: "register.succeeded", Outcome: "success", TargetID: userRef(user.ID), CorrelationID: "registration", At: now})
	})
	if err == nil && s.mailer != nil {
		err = s.mailer.Send(ctx, "email-verification", verificationToken)
	}
	return user, err
}

func (s *Service) Login(ctx context.Context, input LoginInput) (Credentials, domain.User, []domain.RolePermission, error) {
	if err := validateCredentials(input.Email, input.Password); err != nil {
		return Credentials{}, domain.User{}, nil, domain.ErrInvalidCredentials
	}
	if err := s.allow(ctx, "login:"+identityKey(domain.NormalizeEmail(input.Email))); err != nil {
		return Credentials{}, domain.User{}, nil, err
	}
	user, err := s.store.FindUserByEmail(ctx, domain.NormalizeEmail(input.Email))
	if err != nil || user.Status != domain.UserActive || !s.passwords.Verify(input.Password, user.PasswordHash) {
		return Credentials{}, domain.User{}, nil, domain.ErrInvalidCredentials
	}
	roles, err := s.store.UserRoles(ctx, user.ID)
	if err != nil {
		return Credentials{}, domain.User{}, nil, err
	}
	credentials, err := s.createSession(ctx, user, roles, input.CorrelationID, "login.succeeded")
	return credentials, user, roles, err
}

func (s *Service) Refresh(ctx context.Context, input RefreshInput) (Credentials, domain.User, []domain.RolePermission, error) {
	if input.RefreshToken == "" {
		return Credentials{}, domain.User{}, nil, domain.ErrInvalidCredentials
	}
	if err := s.allow(ctx, "refresh:"+identityKey(input.RefreshToken)); err != nil {
		return Credentials{}, domain.User{}, nil, err
	}
	session, err := s.store.FindSessionByRefreshHash(ctx, s.tokens.HashOpaque(input.RefreshToken))
	if err != nil {
		return Credentials{}, domain.User{}, nil, domain.ErrInvalidCredentials
	}
	now := s.clock.Now().UTC()
	user, err := s.store.FindUser(ctx, session.UserID)
	if err != nil {
		return Credentials{}, domain.User{}, nil, domain.ErrInvalidCredentials
	}
	if !session.IsUsable(now) || user.Status != domain.UserActive {
		_ = s.store.WithTransaction(ctx, func(tx Transaction) error { return tx.RevokeSessionFamily(ctx, session.FamilyID, now, "refresh_reuse") })
		return Credentials{}, domain.User{}, nil, domain.ErrInvalidCredentials
	}
	roles, err := s.store.UserRoles(ctx, user.ID)
	if err != nil {
		return Credentials{}, domain.User{}, nil, err
	}
	refresh, refreshHash, err := s.tokens.NewOpaque(32)
	if err != nil {
		return Credentials{}, domain.User{}, nil, err
	}
	replacementID := uuid.New()
	access, err := s.tokens.Sign(userRef(user.ID), s.humanAudience, "human", replacementID.String(), roleNames(roles), scopeNames(roles), accessTokenTTL)
	if err != nil {
		return Credentials{}, domain.User{}, nil, err
	}
	newSession := domain.Session{ID: replacementID, FamilyID: session.FamilyID, UserID: user.ID, Status: domain.SessionActive, ExpiresAt: now.Add(refreshTokenTTL), CreatedAt: now, LastUsedAt: &now}
	reuseDetected := false
	err = s.store.WithTransaction(ctx, func(tx Transaction) error {
		rotated, err := tx.RotateSession(ctx, session.ID, replacementID, now)
		if err != nil {
			return err
		}
		if rotated != 1 {
			reuseDetected = true
			return nil
		}
		if err := tx.CreateSession(ctx, newSession, refreshHash, nil, nil); err != nil {
			return err
		}
		return s.recordAudit(ctx, tx, auditInput{Action: "refresh.succeeded", Outcome: "success", ActorID: userRef(user.ID), ActorType: "user", TargetID: userRef(session.ID), CorrelationID: input.CorrelationID, At: now})
	})
	if err != nil {
		return Credentials{}, domain.User{}, nil, err
	}
	if reuseDetected {
		_ = s.store.WithTransaction(ctx, func(tx Transaction) error { return tx.RevokeSessionFamily(ctx, session.FamilyID, now, "refresh_reuse") })
		return Credentials{}, domain.User{}, nil, domain.ErrInvalidCredentials
	}
	return Credentials{AccessToken: access, RefreshToken: refresh, AccessTokenExpiresIn: int(accessTokenTTL.Seconds()), RefreshExpiresIn: int(refreshTokenTTL.Seconds())}, user, roles, nil
}

func (s *Service) createSession(ctx context.Context, user domain.User, roles []domain.RolePermission, correlationID, action string) (Credentials, error) {
	now := s.clock.Now().UTC()
	sessionID, familyID := uuid.New(), uuid.New()
	refresh, refreshHash, err := s.tokens.NewOpaque(32)
	if err != nil {
		return Credentials{}, err
	}
	access, err := s.tokens.Sign(userRef(user.ID), s.humanAudience, "human", sessionID.String(), roleNames(roles), scopeNames(roles), accessTokenTTL)
	if err != nil {
		return Credentials{}, err
	}
	session := domain.Session{ID: sessionID, FamilyID: familyID, UserID: user.ID, Status: domain.SessionActive, ExpiresAt: now.Add(refreshTokenTTL), CreatedAt: now, LastUsedAt: &now}
	err = s.store.WithTransaction(ctx, func(tx Transaction) error {
		if err := tx.CreateSession(ctx, session, refreshHash, nil, nil); err != nil {
			return err
		}
		return s.recordAudit(ctx, tx, auditInput{Action: action, Outcome: "success", ActorID: userRef(user.ID), ActorType: "user", TargetID: userRef(session.ID), CorrelationID: correlationID, At: now})
	})
	if err != nil {
		return Credentials{}, err
	}
	return Credentials{AccessToken: access, RefreshToken: refresh, AccessTokenExpiresIn: int(accessTokenTTL.Seconds()), RefreshExpiresIn: int(refreshTokenTTL.Seconds())}, nil
}

func (s *Service) Logout(ctx context.Context, userID, sessionID uuid.UUID, correlationID string) error {
	now := s.clock.Now().UTC()
	return s.store.WithTransaction(ctx, func(tx Transaction) error {
		if err := tx.RevokeSession(ctx, sessionID, now, "logout"); err != nil {
			return err
		}
		return s.recordAudit(ctx, tx, auditInput{Action: "logout.succeeded", Outcome: "success", ActorID: userRef(userID), ActorType: "user", TargetID: userRef(sessionID), CorrelationID: correlationID, At: now})
	})
}

func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID, correlationID string) error {
	now := s.clock.Now().UTC()
	return s.store.WithTransaction(ctx, func(tx Transaction) error {
		if err := tx.RevokeUserSessions(ctx, userID, now, "logout_all"); err != nil {
			return err
		}
		return s.recordAudit(ctx, tx, auditInput{Action: "logout_all.succeeded", Outcome: "success", ActorID: userRef(userID), ActorType: "user", TargetID: userRef(userID), CorrelationID: correlationID, At: now})
	})
}

func (s *Service) Sessions(ctx context.Context, userID uuid.UUID) ([]domain.Session, error) {
	return s.store.ListSessions(ctx, userID)
}

func (s *Service) User(ctx context.Context, userID uuid.UUID) (domain.User, []domain.RolePermission, error) {
	user, err := s.store.FindUser(ctx, userID)
	if err != nil {
		return domain.User{}, nil, err
	}
	roles, err := s.store.UserRoles(ctx, userID)
	return user, roles, err
}

func (s *Service) ChangeUserStatus(ctx context.Context, targetID uuid.UUID, status domain.UserStatus, actorID uuid.UUID, correlationID string) error {
	if status != domain.UserActive && status != domain.UserDisabled {
		return domain.ErrConflict
	}
	now := s.clock.Now().UTC()
	return s.store.WithTransaction(ctx, func(tx Transaction) error {
		if err := tx.UpdateUserStatus(ctx, targetID, status, now); err != nil {
			return err
		}
		if status == domain.UserDisabled {
			if err := tx.RevokeUserSessions(ctx, targetID, now, "user_disabled"); err != nil {
				return err
			}
		}
		action := "user.enabled"
		if status == domain.UserDisabled {
			action = "user.disabled"
		}
		return s.recordAudit(ctx, tx, auditInput{Action: action, Outcome: "success", ActorID: userRef(actorID), ActorType: "user", TargetID: userRef(targetID), CorrelationID: correlationID, At: now})
	})
}

func (s *Service) AssignRole(ctx context.Context, targetID, actorID uuid.UUID, role, correlationID string) error {
	if err := s.store.FindRole(ctx, role); err != nil {
		return err
	}
	now := s.clock.Now().UTC()
	return s.store.WithTransaction(ctx, func(tx Transaction) error {
		if err := tx.AssignRole(ctx, targetID, role, now, uuid.NullUUID{UUID: actorID, Valid: true}); err != nil {
			return err
		}
		return s.recordAudit(ctx, tx, auditInput{Action: "role.assigned", Outcome: "success", ActorID: userRef(actorID), ActorType: "user", TargetID: userRef(targetID), CorrelationID: correlationID, At: now, Role: role})
	})
}

func (s *Service) RemoveRole(ctx context.Context, targetID, actorID uuid.UUID, role, correlationID string) error {
	now := s.clock.Now().UTC()
	return s.store.WithTransaction(ctx, func(tx Transaction) error {
		if err := tx.RemoveRole(ctx, targetID, role); err != nil {
			return err
		}
		return s.recordAudit(ctx, tx, auditInput{Action: "role.removed", Outcome: "success", ActorID: userRef(actorID), ActorType: "user", TargetID: userRef(targetID), CorrelationID: correlationID, At: now, Role: role})
	})
}

func (s *Service) RequestPasswordReset(ctx context.Context, email, correlationID string) error {
	if err := s.allow(ctx, "reset:"+identityKey(domain.NormalizeEmail(email))); err != nil {
		return err
	}
	user, err := s.store.FindUserByEmail(ctx, domain.NormalizeEmail(email))
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	token, hash, err := s.tokens.NewOpaque(32)
	if err != nil {
		return err
	}
	now := s.clock.Now().UTC()
	expires := now.Add(resetTokenTTL)
	err = s.store.WithTransaction(ctx, func(tx Transaction) error {
		if err := tx.CreateResetToken(ctx, uuid.New(), user.ID, hash, expires, now); err != nil {
			return err
		}
		return s.recordAudit(ctx, tx, auditInput{Action: "password_reset.requested", Outcome: "success", TargetID: userRef(user.ID), CorrelationID: correlationID, At: now})
	})
	if err == nil && s.mailer != nil {
		err = s.mailer.Send(ctx, "password-reset", token)
	}
	return err
}

func (s *Service) ConfirmPasswordReset(ctx context.Context, token, password, correlationID string) error {
	if len(password) < 12 || len(password) > 128 {
		return domain.ErrConflict
	}
	reset, err := s.store.FindResetToken(ctx, s.tokens.HashOpaque(token))
	if err != nil || !reset.IsUsable(s.clock.Now()) {
		return domain.ErrInvalidCredentials
	}
	hash, err := s.passwords.Hash(password)
	if err != nil {
		return err
	}
	now := s.clock.Now().UTC()
	return s.store.WithTransaction(ctx, func(tx Transaction) error {
		if err := tx.UpdatePassword(ctx, reset.UserID, hash, now); err != nil {
			return err
		}
		if err := tx.RevokeUserSessions(ctx, reset.UserID, now, "password_reset"); err != nil {
			return err
		}
		consumed, err := tx.ConsumeResetToken(ctx, reset.ID, now)
		if err != nil {
			return err
		}
		if consumed != 1 {
			return domain.ErrAlreadyUsed
		}
		return s.recordAudit(ctx, tx, auditInput{Action: "password_reset.completed", Outcome: "success", TargetID: userRef(reset.UserID), CorrelationID: correlationID, At: now})
	})
}

func (s *Service) ConfirmEmail(ctx context.Context, token, correlationID string) error {
	verification, err := s.store.FindVerificationToken(ctx, s.tokens.HashOpaque(token))
	if err != nil || !verification.IsUsable(s.clock.Now()) {
		return domain.ErrInvalidCredentials
	}
	now := s.clock.Now().UTC()
	return s.store.WithTransaction(ctx, func(tx Transaction) error {
		consumed, err := tx.ConsumeVerificationToken(ctx, verification.ID, now)
		if err != nil {
			return err
		}
		if consumed != 1 {
			return domain.ErrAlreadyUsed
		}
		if err := tx.MarkEmailVerified(ctx, verification.UserID, now); err != nil {
			return err
		}
		return s.recordAudit(ctx, tx, auditInput{Action: "email_verification.completed", Outcome: "success", TargetID: userRef(verification.UserID), CorrelationID: correlationID, At: now})
	})
}

func (s *Service) ServiceToken(ctx context.Context, clientID, secret string, requested []string, correlationID string) (MachineCredentials, error) {
	if strings.TrimSpace(clientID) == "" || secret == "" || len(requested) == 0 {
		return MachineCredentials{}, domain.ErrInvalidCredentials
	}
	if err := s.allow(ctx, "service-token:"+identityKey(clientID)); err != nil {
		return MachineCredentials{}, err
	}
	principal, err := s.store.FindServicePrincipal(ctx, clientID)
	if err != nil || principal.Status != "ACTIVE" {
		return MachineCredentials{}, domain.ErrInvalidCredentials
	}
	credentials, err := s.store.ActiveServiceCredentials(ctx, principal.ID, s.clock.Now())
	if err != nil {
		return MachineCredentials{}, domain.ErrInvalidCredentials
	}
	valid := false
	for _, credential := range credentials {
		if s.passwords.Verify(secret, credential.SecretHash) {
			valid = true
			break
		}
	}
	if !valid {
		return MachineCredentials{}, domain.ErrInvalidCredentials
	}
	grant := make(map[string]struct{}, len(principal.Scopes))
	for _, scope := range principal.Scopes {
		grant[scope] = struct{}{}
	}
	for _, scope := range requested {
		if _, ok := grant[scope]; !ok {
			return MachineCredentials{}, domain.ErrForbidden
		}
	}
	access, err := s.tokens.Sign("svc_"+principal.ClientID, principal.Audience, "service", "", nil, requested, machineTokenTTL)
	if err != nil {
		return MachineCredentials{}, err
	}
	now := s.clock.Now().UTC()
	err = s.store.WithTransaction(ctx, func(tx Transaction) error {
		if err := tx.MarkServicePrincipalUsed(ctx, principal.ID, now); err != nil {
			return err
		}
		return s.recordAudit(ctx, tx, auditInput{Action: "service_token.issued", Outcome: "success", ActorID: "svc_" + principal.ClientID, ActorType: "service", CorrelationID: correlationID, At: now})
	})
	if err != nil {
		return MachineCredentials{}, err
	}
	return MachineCredentials{AccessToken: access, ExpiresIn: int(machineTokenTTL.Seconds())}, nil
}

type auditInput struct {
	Action, Outcome, ActorID, ActorType, TargetID, CorrelationID, Role string
	At                                                                 time.Time
}

func (s *Service) recordAudit(ctx context.Context, tx Transaction, input auditInput) error {
	if input.CorrelationID == "" {
		input.CorrelationID = "identity"
	}
	aggregateID := "aud_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	eventID := "evt_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	data := map[string]any{"action": input.Action, "outcome": input.Outcome, "actor_id": nullableValue(input.ActorID), "actor_type": nullableValue(input.ActorType), "target_type": nullableValue("user"), "target_id": nullableValue(input.TargetID), "reason_code": nil, "role": nullableValue(input.Role)}
	payload, err := json.Marshal(map[string]any{"event_id": eventID, "event_type": "commerce.security.audit.v1", "aggregate_type": "security_audit", "aggregate_id": aggregateID, "correlation_id": input.CorrelationID, "causation_id": nil, "occurred_at": input.At.Format(time.RFC3339Nano), "producer": "identity", "schema_version": 1, "data": data})
	if err != nil {
		return err
	}
	auditID := uuid.New()
	actorID, actorType, targetID := optional(input.ActorID), optional(input.ActorType), optional(input.TargetID)
	if err := tx.RecordAudit(ctx, AuditRecord{ID: auditID, ActorID: actorID, ActorType: actorType, Action: input.Action, TargetType: optional("user"), TargetID: targetID, Metadata: []byte(`{}`), CorrelationID: input.CorrelationID, OccurredAt: input.At}); err != nil {
		return err
	}
	return tx.RecordOutbox(ctx, OutboxRecord{ID: uuid.New(), EventType: "commerce.security.audit.v1", AggregateType: "security_audit", AggregateID: aggregateID, Payload: payload, Headers: []byte(`{}`), CreatedAt: input.At})
}

func (s *Service) allow(ctx context.Context, key string) error {
	if s.limiter == nil {
		return nil
	}
	allowed, err := s.limiter.Allow(ctx, key)
	if err != nil {
		return domain.ErrDependency
	}
	if !allowed {
		return domain.ErrRateLimited
	}
	return nil
}
func validateCredentials(email, password string) error {
	normalized := domain.NormalizeEmail(email)
	parsed, parseErr := mail.ParseAddress(normalized)
	if parseErr != nil || parsed.Address != normalized || len(normalized) > 320 || len(password) < 12 || len(password) > 128 {
		return domain.ErrConflict
	}
	return nil
}
func identityKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func userRef(id uuid.UUID) string { return "usr_" + strings.ReplaceAll(id.String(), "-", "") }
func roleNames(values []domain.RolePermission) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, value := range values {
		if _, ok := seen[value.Role]; !ok {
			seen[value.Role] = struct{}{}
			result = append(result, value.Role)
		}
	}
	sort.Strings(result)
	return result
}
func scopeNames(values []domain.RolePermission) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, value := range values {
		if _, ok := seen[value.Scope]; !ok {
			seen[value.Scope] = struct{}{}
			result = append(result, value.Scope)
		}
	}
	sort.Strings(result)
	return result
}
func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
func nullableValue(value string) any {
	if value == "" {
		return nil
	}
	return value
}
