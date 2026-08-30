package identity

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	identitycrypto "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/adapters/outbound/crypto"
	domain "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/domain/identity"
	"github.com/google/uuid"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type allowAll struct{}

func (allowAll) Allow(context.Context, string) (bool, error) { return true, nil }

type rejectLimiter struct{}

func (rejectLimiter) Allow(context.Context, string) (bool, error) { return false, nil }

type errorLimiter struct{}

func (errorLimiter) Allow(context.Context, string) (bool, error) {
	return false, errors.New("redis unavailable")
}

type dimensionLimiter struct {
	allowed bool
	err     error
	keys    []string
}

func (l *dimensionLimiter) Allow(context.Context, string) (bool, error) { return l.allowed, l.err }
func (l *dimensionLimiter) AllowMany(_ context.Context, keys []string) (bool, error) {
	l.keys = append([]string(nil), keys...)
	return l.allowed, l.err
}

type recordingMailer struct {
	sent  bool
	token string
}

func (m *recordingMailer) Send(_ context.Context, _ string, _ string, token string) error {
	m.sent = true
	m.token = token
	return nil
}

type failingMailer struct{}

func (failingMailer) Send(context.Context, string, string, string) error {
	return errors.New("mailer unavailable")
}

type fakeStore struct {
	users                 map[uuid.UUID]domain.User
	byEmail               map[string]uuid.UUID
	sessions              map[uuid.UUID]domain.Session
	roles                 map[uuid.UUID][]domain.RolePermission
	reset                 map[string]domain.OneTimeToken
	audits                int
	outbox                int
	resetToken            domain.OneTimeToken
	verificationToken     domain.OneTimeToken
	principal             domain.ServicePrincipal
	credentials           []domain.ServiceCredential
	consumeCount          int64
	rolesErr              error
	sessionErr            error
	transactionErr        error
	verificationCreateErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{users: map[uuid.UUID]domain.User{}, byEmail: map[string]uuid.UUID{}, sessions: map[uuid.UUID]domain.Session{}, roles: map[uuid.UUID][]domain.RolePermission{}, reset: map[string]domain.OneTimeToken{}, consumeCount: 1}
}
func (s *fakeStore) FindUserByEmail(_ context.Context, email string) (domain.User, error) {
	id, ok := s.byEmail[email]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return s.users[id], nil
}
func (s *fakeStore) FindUser(_ context.Context, id uuid.UUID) (domain.User, error) {
	user, ok := s.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return user, nil
}
func (s *fakeStore) UserRoles(_ context.Context, id uuid.UUID) ([]domain.RolePermission, error) {
	if s.rolesErr != nil {
		return nil, s.rolesErr
	}
	return append([]domain.RolePermission(nil), s.roles[id]...), nil
}
func (s *fakeStore) FindRole(_ context.Context, name string) error {
	if name == "customer" || name == "admin" {
		return nil
	}
	return domain.ErrNotFound
}
func (s *fakeStore) FindSessionByRefreshHash(context.Context, []byte) (domain.Session, error) {
	if s.sessionErr != nil {
		return domain.Session{}, s.sessionErr
	}
	for _, session := range s.sessions {
		return session, nil
	}
	return domain.Session{}, domain.ErrNotFound
}
func (s *fakeStore) FindSessionByID(_ context.Context, id uuid.UUID) (domain.Session, error) {
	if s.sessionErr != nil {
		return domain.Session{}, s.sessionErr
	}
	session, ok := s.sessions[id]
	if !ok {
		return domain.Session{}, domain.ErrNotFound
	}
	return session, nil
}
func (s *fakeStore) ListSessions(_ context.Context, id uuid.UUID) ([]domain.Session, error) {
	result := []domain.Session{}
	for _, session := range s.sessions {
		if session.UserID == id {
			result = append(result, session)
		}
	}
	return result, nil
}
func (s *fakeStore) ListUsersPage(_ context.Context, query UserPageQuery) (UserPage, error) {
	page := UserPage{}
	for id, user := range s.users {
		if query.Status != "" && string(user.Status) != query.Status {
			continue
		}
		if query.EmailPrefix != "" && !strings.HasPrefix(user.EmailNormalized, query.EmailPrefix) {
			continue
		}
		roles := roleNames(s.roles[id])
		if query.Role != "" {
			found := false
			for _, role := range roles {
				if role == query.Role {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		page.Items = append(page.Items, UserPageItem{User: user, Roles: roles})
	}
	if int32(len(page.Items)) > query.Limit {
		page.HasMore = true
		page.Items = page.Items[:query.Limit]
	}
	return page, nil
}
func (s *fakeStore) ListSessionsPage(_ context.Context, id uuid.UUID, query SessionPageQuery) (SessionPage, error) {
	sessions, err := s.ListSessions(context.Background(), id)
	if err != nil {
		return SessionPage{}, err
	}
	page := SessionPage{Items: sessions}
	if int32(len(page.Items)) > query.Limit {
		page.HasMore = true
		page.Items = page.Items[:query.Limit]
	}
	if len(page.Items) > 0 {
		page.LastCreated, page.LastID = page.Items[len(page.Items)-1].CreatedAt, page.Items[len(page.Items)-1].ID
	}
	return page, nil
}
func (s *fakeStore) FindServicePrincipal(context.Context, string) (domain.ServicePrincipal, error) {
	if s.principal.ID == uuid.Nil {
		return domain.ServicePrincipal{}, domain.ErrNotFound
	}
	return s.principal, nil
}
func (s *fakeStore) ActiveServiceCredentials(context.Context, uuid.UUID, time.Time) ([]domain.ServiceCredential, error) {
	return s.credentials, nil
}
func (s *fakeStore) FindResetToken(context.Context, []byte) (domain.OneTimeToken, error) {
	if s.resetToken.ID == uuid.Nil {
		return domain.OneTimeToken{}, domain.ErrNotFound
	}
	return s.resetToken, nil
}
func (s *fakeStore) FindVerificationToken(context.Context, []byte) (domain.OneTimeToken, error) {
	if s.verificationToken.ID == uuid.Nil {
		return domain.OneTimeToken{}, domain.ErrNotFound
	}
	return s.verificationToken, nil
}
func (s *fakeStore) WithTransaction(_ context.Context, fn func(Transaction) error) error {
	if s.transactionErr != nil {
		return s.transactionErr
	}
	return fn((*fakeTransaction)(s))
}

type fakeTransaction fakeStore

func (t *fakeTransaction) CreateUser(_ context.Context, user domain.User) (domain.User, error) {
	if _, ok := t.byEmail[user.EmailNormalized]; ok {
		return domain.User{}, domain.ErrConflict
	}
	t.users[user.ID] = user
	t.byEmail[user.EmailNormalized] = user.ID
	return user, nil
}
func (t *fakeTransaction) CreateSession(_ context.Context, session domain.Session, refresh, userAgent, ipPrefix []byte) error {
	t.sessions[session.ID] = session
	_, _, _ = refresh, userAgent, ipPrefix
	return nil
}
func (t *fakeTransaction) RotateSession(_ context.Context, id, replacement uuid.UUID, now time.Time) (int64, error) {
	session, ok := t.sessions[id]
	if !ok || session.Status != domain.SessionActive {
		return 0, nil
	}
	session.Status = domain.SessionRotated
	session.LastUsedAt = &now
	t.sessions[id] = session
	_ = replacement
	return 1, nil
}
func (t *fakeTransaction) RevokeSessionFamily(_ context.Context, family uuid.UUID, now time.Time, _ string) error {
	for id, session := range t.sessions {
		if session.FamilyID == family {
			session.Status = domain.SessionRevoked
			session.RevokedAt = &now
			t.sessions[id] = session
		}
	}
	return nil
}
func (t *fakeTransaction) RevokeSession(_ context.Context, id uuid.UUID, now time.Time, _ string) error {
	session, ok := t.sessions[id]
	if ok {
		session.Status = domain.SessionRevoked
		session.RevokedAt = &now
		t.sessions[id] = session
	}
	return nil
}
func (t *fakeTransaction) RevokeUserSessions(_ context.Context, user uuid.UUID, now time.Time, _ string) error {
	for id, session := range t.sessions {
		if session.UserID == user {
			session.Status = domain.SessionRevoked
			session.RevokedAt = &now
			t.sessions[id] = session
		}
	}
	return nil
}
func (t *fakeTransaction) UpdateUserStatus(_ context.Context, id uuid.UUID, status domain.UserStatus, now time.Time) error {
	user := t.users[id]
	user.Status, user.UpdatedAt = status, now
	t.users[id] = user
	return nil
}
func (t *fakeTransaction) UpdatePassword(_ context.Context, id uuid.UUID, hash string, now time.Time) error {
	user := t.users[id]
	user.PasswordHash, user.UpdatedAt = hash, now
	t.users[id] = user
	return nil
}
func (t *fakeTransaction) AssignRole(_ context.Context, id uuid.UUID, role string, _ time.Time, _ uuid.NullUUID) error {
	t.roles[id] = append(t.roles[id], domain.RolePermission{Role: role, Scope: role + ":scope"})
	return nil
}
func (t *fakeTransaction) RemoveRole(context.Context, uuid.UUID, string) error { return nil }
func (t *fakeTransaction) CreateResetToken(_ context.Context, id, user uuid.UUID, hash []byte, expires, _ time.Time) error {
	t.reset[string(hash)] = domain.OneTimeToken{ID: id, UserID: user, ExpiresAt: expires}
	t.resetToken = domain.OneTimeToken{ID: id, UserID: user, ExpiresAt: expires}
	return nil
}
func (t *fakeTransaction) ConsumeResetToken(context.Context, uuid.UUID, time.Time) (int64, error) {
	return t.consumeCount, nil
}
func (t *fakeTransaction) CreateVerificationToken(_ context.Context, id, user uuid.UUID, _ []byte, expires, _ time.Time) error {
	if t.verificationCreateErr != nil {
		return t.verificationCreateErr
	}
	t.verificationToken = domain.OneTimeToken{ID: id, UserID: user, ExpiresAt: expires}
	return nil
}
func (t *fakeTransaction) ConsumeVerificationToken(context.Context, uuid.UUID, time.Time) (int64, error) {
	return t.consumeCount, nil
}
func (t *fakeTransaction) MarkEmailVerified(_ context.Context, id uuid.UUID, now time.Time) error {
	user := t.users[id]
	user.EmailVerifiedAt, user.Status, user.UpdatedAt = &now, domain.UserActive, now
	t.users[id] = user
	return nil
}
func (t *fakeTransaction) MarkServicePrincipalUsed(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (t *fakeTransaction) RecordAudit(context.Context, AuditRecord) error   { t.audits++; return nil }
func (t *fakeTransaction) RecordOutbox(context.Context, OutboxRecord) error { t.outbox++; return nil }

func newServiceForTest(t *testing.T, store *fakeStore, limiter Limiter) *Service {
	t.Helper()
	signer, err := identitycrypto.GenerateSigner("test", "identity.test")
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(Options{Store: store, Passwords: identitycrypto.NewPasswordHasher(), Tokens: identitycrypto.NewTokenServiceForSigner(signer), Clock: fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, Limiter: limiter, HumanAudience: "platform-api", MachineAudience: "platform-internal"})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func registerAndVerify(t *testing.T, service *Service, email string) domain.User {
	t.Helper()
	mailer := &recordingMailer{}
	service.mailer = mailer
	user, err := service.Register(context.Background(), RegisterInput{Email: email, Password: "correct horse battery staple"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ConfirmEmail(context.Background(), mailer.token, "registration-test"); err != nil {
		t.Fatal(err)
	}
	return user
}

func TestRegisterLoginRefreshAndLogout(t *testing.T) {
	store := newFakeStore()
	service := newServiceForTest(t, store, allowAll{})
	registrationMailer := &recordingMailer{}
	service.mailer = registrationMailer
	ctx := context.Background()
	if _, err := service.Register(ctx, RegisterInput{Email: "person@example.test", Password: "correct horse battery staple"}); err != nil {
		t.Fatal(err)
	}
	if !registrationMailer.sent {
		t.Fatal("verification mail was not sent")
	}
	if err := service.ConfirmEmail(ctx, registrationMailer.token, "correlation"); err != nil {
		t.Fatal(err)
	}
	credentials, user, _, err := service.Login(ctx, LoginInput{Email: "PERSON@example.test", Password: "correct horse battery staple"})
	if err != nil {
		t.Fatal(err)
	}
	if credentials.AccessToken == "" || credentials.RefreshToken == "" {
		t.Fatal("missing credentials")
	}
	if _, _, _, err := service.Login(ctx, LoginInput{Email: "person@example.test", Password: "wrong password"}); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v", err)
	}
	refreshed, _, _, err := service.Refresh(ctx, RefreshInput{RefreshToken: credentials.RefreshToken})
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.RefreshToken == credentials.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}
	if err := service.LogoutAll(ctx, user.ID, "correlation"); err != nil {
		t.Fatal(err)
	}
	if store.audits == 0 || store.outbox == 0 {
		t.Fatal("audit/outbox not recorded")
	}
}

func TestRegistrationRateLimitAndValidation(t *testing.T) {
	store := newFakeStore()
	service := newServiceForTest(t, store, rejectLimiter{})
	if _, err := service.Register(context.Background(), RegisterInput{Email: "x@example.test", Password: "short"}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("got %v", err)
	}
	if _, err := service.Register(context.Background(), RegisterInput{Email: "x@example.test", Password: "correct horse battery staple"}); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("got %v", err)
	}
}

func TestSourceAwareLimiterUsesAtomicDimensions(t *testing.T) {
	store := newFakeStore()
	limiter := &dimensionLimiter{allowed: true}
	service := newServiceForTest(t, store, limiter)
	if _, err := service.Register(context.Background(), RegisterInput{Email: "source@example.test", Password: "correct horse battery staple", SourceKey: "192.0.2.0/24"}); err != nil {
		t.Fatal(err)
	}
	if len(limiter.keys) != 3 || !strings.Contains(limiter.keys[0], "register-account:") || !strings.Contains(limiter.keys[1], "register-global:") || !strings.Contains(limiter.keys[2], "register-source:") {
		t.Fatalf("unexpected limiter dimensions: %v", limiter.keys)
	}
	limiter.allowed = false
	if _, err := service.Register(context.Background(), RegisterInput{Email: "blocked@example.test", Password: "correct horse battery staple", SourceKey: "192.0.2.0/24"}); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("blocked source: %v", err)
	}
	limiter.err = errors.New("redis unavailable")
	if _, err := service.Register(context.Background(), RegisterInput{Email: "error@example.test", Password: "correct horse battery staple", SourceKey: "192.0.2.0/24"}); !errors.Is(err, domain.ErrDependency) {
		t.Fatalf("limiter failure: %v", err)
	}
}

func TestAdministrativeRecoveryAndMachineFlows(t *testing.T) {
	store := newFakeStore()
	service := newServiceForTest(t, store, allowAll{})
	ctx := context.Background()
	user, err := service.Register(ctx, RegisterInput{Email: "admin@example.test", Password: "correct horse battery staple"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Register(ctx, RegisterInput{Email: "ADMIN@example.test", Password: "correct horse battery staple"}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate registration: %v", err)
	}
	if _, _, err := service.User(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.AssignRole(ctx, user.ID, user.ID, "admin", "corr"); err != nil {
		t.Fatal(err)
	}
	if err := service.RemoveRole(ctx, user.ID, user.ID, "admin", "corr"); err != nil {
		t.Fatal(err)
	}
	if err := service.ChangeUserStatus(ctx, user.ID, domain.UserDisabled, user.ID, "corr"); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := service.Login(ctx, LoginInput{Email: user.Email, Password: "correct horse battery staple"}); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("disabled user login: %v", err)
	}
	if err := service.ChangeUserStatus(ctx, user.ID, domain.UserActive, user.ID, "corr"); err != nil {
		t.Fatal(err)
	}
	if err := service.ChangeUserStatus(ctx, user.ID, domain.UserStatus("unknown"), user.ID, "corr"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("got %v", err)
	}
	if err := service.RequestPasswordReset(ctx, "missing@example.test", "corr"); err != nil {
		t.Fatal(err)
	}
	if err := service.RequestPasswordResetFromSource(ctx, "missing-source@example.test", "192.0.2.0/24", "corr"); err != nil {
		t.Fatal(err)
	}
	if err := service.RequestPasswordReset(ctx, user.Email, "corr"); err != nil {
		t.Fatal(err)
	}
	service.mailer = failingMailer{}
	if err := service.RequestPasswordReset(ctx, user.Email, "corr"); err == nil {
		t.Fatal("mailer failure was ignored")
	}
	mailer := &recordingMailer{}
	service.mailer = mailer
	if err := service.RequestPasswordReset(ctx, user.Email, "corr"); err != nil || !mailer.sent {
		t.Fatalf("simulated mail was not sent: %v", err)
	}
	store.resetToken = domain.OneTimeToken{ID: uuid.New(), UserID: user.ID, ExpiresAt: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)}
	if err := service.ConfirmPasswordReset(ctx, "opaque-reset-token", "new correct horse battery", "corr"); err != nil {
		t.Fatal(err)
	}
	store.verificationToken = domain.OneTimeToken{ID: uuid.New(), UserID: user.ID, ExpiresAt: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)}
	if err := service.ConfirmEmail(ctx, "opaque-verification-token", "corr"); err != nil {
		t.Fatal(err)
	}
	store.resetToken.ExpiresAt = time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	store.consumeCount = 0
	if err := service.ConfirmPasswordReset(ctx, "opaque-reset-token", "another correct horse battery", "corr"); !errors.Is(err, domain.ErrAlreadyUsed) {
		t.Fatalf("raced reset: %v", err)
	}
	store.verificationToken.ExpiresAt = time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := service.ConfirmEmail(ctx, "opaque-verification-token", "corr"); !errors.Is(err, domain.ErrAlreadyUsed) {
		t.Fatalf("raced verification: %v", err)
	}
	store.principal = domain.ServicePrincipal{ID: uuid.New(), ClientID: "orders", Status: "ACTIVE", Audience: "platform-internal", Scopes: []string{"orders:read"}}
	hasher := identitycrypto.NewPasswordHasher()
	secretHash, err := hasher.Hash("secret")
	if err != nil {
		t.Fatal(err)
	}
	store.credentials = []domain.ServiceCredential{{ID: uuid.New(), ServicePrincipalID: store.principal.ID, SecretHash: secretHash}}
	if _, err := service.ServiceToken(ctx, "orders", "secret", []string{"orders:write"}, "corr"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("got %v", err)
	}
	if _, err := service.ServiceToken(ctx, "orders", "wrong-secret", []string{"orders:read"}, "corr"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("bad machine secret: %v", err)
	}
	if result, err := service.ServiceToken(ctx, "orders", "secret", []string{"orders:read"}, "corr"); err != nil || result.AccessToken == "" {
		t.Fatalf("machine token: %v", err)
	}
	if result, err := service.ServiceTokenFromSource(ctx, "orders", "secret", []string{"orders:read"}, "192.0.2.0/24", "corr"); err != nil || result.AccessToken == "" {
		t.Fatalf("source machine token: %v", err)
	}
	store.principal.Status = "DISABLED"
	if _, err := service.ServiceToken(ctx, "orders", "secret", []string{"orders:read"}, "corr"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("disabled principal: %v", err)
	}
	if _, _, err := service.User(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing user: %v", err)
	}
	if err := service.AssignRole(ctx, user.ID, user.ID, "unknown", "corr"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown role: %v", err)
	}
}

func TestServiceConstructionAndLimiterFailures(t *testing.T) {
	if _, err := NewService(Options{}); err == nil {
		t.Fatal("expected incomplete dependency error")
	}
	store := newFakeStore()
	service := newServiceForTest(t, store, errorLimiter{})
	if _, err := service.Register(context.Background(), RegisterInput{Email: "a@example.test", Password: "correct horse battery staple"}); !errors.Is(err, domain.ErrDependency) {
		t.Fatalf("got %v", err)
	}
	if _, _, _, err := service.Login(context.Background(), LoginInput{Email: "missing@example.test", Password: "correct horse battery staple"}); !errors.Is(err, domain.ErrDependency) {
		t.Fatalf("limiter failure: %v", err)
	}
	signer, err := identitycrypto.GenerateSigner("test", "identity.test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(Options{Store: store, Passwords: identitycrypto.NewPasswordHasher(), Tokens: identitycrypto.NewTokenServiceForSigner(signer), Clock: fixedClock{now: time.Now()}}); err == nil {
		t.Fatal("expected audience validation error")
	}
	store.transactionErr = errors.New("transaction unavailable")
	service = newServiceForTest(t, store, nil)
	if _, err := service.Register(context.Background(), RegisterInput{Email: "transaction@example.test", Password: "correct horse battery staple"}); err == nil {
		t.Fatal("transaction error was ignored")
	}
	store.transactionErr = nil
	store.verificationCreateErr = errors.New("verification token storage unavailable")
	if _, err := service.Register(context.Background(), RegisterInput{Email: "verification@example.test", Password: "correct horse battery staple"}); err == nil {
		t.Fatal("verification storage error was ignored")
	}
}

func TestValidationHelpersAndFailureBranches(t *testing.T) {
	for _, value := range []string{"", "not-an-email", "person@example.test"} {
		err := validateCredentials(value, "correct horse battery staple")
		if value == "person@example.test" && err != nil {
			t.Fatalf("valid credentials rejected: %v", err)
		}
		if value != "person@example.test" && err == nil {
			t.Fatalf("invalid email accepted: %q", value)
		}
	}
	if validateCredentials("person@example.test", "short") == nil {
		t.Fatal("short password accepted")
	}
	if identityKey("same") != identityKey("same") || identityKey("same") == identityKey("other") {
		t.Fatal("identity key is not stable")
	}
	if len(roleNames([]domain.RolePermission{{Role: "b"}, {Role: "a"}, {Role: "b"}})) != 2 {
		t.Fatal("role normalization failed")
	}
	if len(scopeNames([]domain.RolePermission{{Scope: "z"}, {Scope: "z"}})) != 1 {
		t.Fatal("scope normalization failed")
	}
	store := newFakeStore()
	service := newServiceForTest(t, store, nil)
	if _, _, _, err := service.Refresh(context.Background(), RefreshInput{}); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("empty refresh: %v", err)
	}
	if err := service.RequestPasswordReset(context.Background(), "missing@example.test", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ServiceToken(context.Background(), "", "", nil, ""); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("empty machine credentials: %v", err)
	}
	if err := service.ConfirmPasswordReset(context.Background(), "token", "short", ""); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("short reset password: %v", err)
	}
}

func TestSessionListingLogoutAndRecoveryOutcomes(t *testing.T) {
	store := newFakeStore()
	service := newServiceForTest(t, store, allowAll{})
	ctx := context.Background()
	user := registerAndVerify(t, service, "recover@example.test")
	credentials, _, _, err := service.Login(ctx, LoginInput{Email: user.Email, Password: "correct horse battery staple"})
	if err != nil {
		t.Fatal(err)
	}
	if sessions, err := service.Sessions(ctx, user.ID); err != nil || len(sessions) != 1 {
		t.Fatalf("sessions: %v", err)
	}
	if err := service.Logout(ctx, user.ID, storeSessionID(store), "corr"); err != nil {
		t.Fatal(err)
	}
	if err := service.RequestPasswordReset(ctx, user.Email, "corr"); err != nil {
		t.Fatal(err)
	}
	if err := service.ConfirmPasswordReset(ctx, "not-the-real-token", "short", "corr"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("short reset: %v", err)
	}
	store.resetToken.ExpiresAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := service.ConfirmPasswordReset(ctx, "not-the-real-token", "new correct horse battery", "corr"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expired reset: %v", err)
	}
	store.verificationToken = domain.OneTimeToken{ID: uuid.New(), UserID: user.ID, ExpiresAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	if err := service.ConfirmEmail(ctx, "expired-token", "corr"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expired verification: %v", err)
	}
	if _, _, _, err := service.Refresh(ctx, RefreshInput{RefreshToken: credentials.RefreshToken}); err == nil {
		t.Fatal("revoked session refreshed")
	}
}

func TestBoundedCollectionQueriesDelegateToCollectionStore(t *testing.T) {
	store := newFakeStore()
	service := newServiceForTest(t, store, nil)
	user := domain.User{ID: uuid.New(), Email: "alice@example.test", EmailNormalized: "alice@example.test", Status: domain.UserActive, CreatedAt: time.Now().UTC()}
	store.users[user.ID], store.byEmail[user.EmailNormalized] = user, user.ID
	store.roles[user.ID] = []domain.RolePermission{{Role: "admin", Scope: "identity:users:read"}}
	page, err := service.UsersPage(context.Background(), UserPageQuery{Role: "admin", Limit: 10})
	if err != nil || len(page.Items) != 1 || page.Items[0].Roles[0] != "admin" {
		t.Fatalf("users page: %+v %v", page, err)
	}
	sessionID := uuid.New()
	store.sessions[sessionID] = domain.Session{ID: sessionID, UserID: user.ID, Status: domain.SessionActive, CreatedAt: user.CreatedAt, ExpiresAt: user.CreatedAt.Add(time.Hour)}
	sessions, err := service.SessionsPage(context.Background(), user.ID, SessionPageQuery{Limit: 10})
	if err != nil || len(sessions.Items) != 1 {
		t.Fatalf("sessions page: %+v %v", sessions, err)
	}
}

func TestIsSessionActiveReflectsPersistentState(t *testing.T) {
	store := newFakeStore()
	service := newServiceForTest(t, store, nil)
	sessionID := uuid.New()
	store.sessions[sessionID] = domain.Session{
		ID:        sessionID,
		Status:    domain.SessionActive,
		ExpiresAt: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	active, err := service.IsSessionActive(context.Background(), sessionID)
	if err != nil || !active {
		t.Fatalf("active session: active=%v err=%v", active, err)
	}
	store.sessions[sessionID] = domain.Session{ID: sessionID, Status: domain.SessionRevoked, ExpiresAt: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)}
	active, err = service.IsSessionActive(context.Background(), sessionID)
	if err != nil || active {
		t.Fatalf("revoked session: active=%v err=%v", active, err)
	}
	if _, err := service.IsSessionActive(context.Background(), uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing session: %v", err)
	}
	store.sessionErr = errors.New("session lookup unavailable")
	if _, err := service.IsSessionActive(context.Background(), sessionID); err == nil {
		t.Fatal("session lookup error was ignored")
	}
}

func storeSessionID(store *fakeStore) uuid.UUID {
	for id := range store.sessions {
		return id
	}
	return uuid.Nil
}

func TestRefreshReuseRevokesSessionFamily(t *testing.T) {
	store := newFakeStore()
	service := newServiceForTest(t, store, allowAll{})
	ctx := context.Background()
	user := registerAndVerify(t, service, "reuse@example.test")
	credentials, _, _, err := service.Login(ctx, LoginInput{Email: user.Email, Password: "correct horse battery staple"})
	if err != nil {
		t.Fatal(err)
	}
	for id, session := range store.sessions {
		session.Status = domain.SessionRotated
		store.sessions[id] = session
	}
	if _, _, _, err := service.Refresh(ctx, RefreshInput{RefreshToken: credentials.RefreshToken}); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("reused refresh: %v", err)
	}
	for _, session := range store.sessions {
		if session.Status != domain.SessionRevoked {
			t.Fatalf("session family was not revoked: %s", session.Status)
		}
	}
}

func TestRefreshAndRoleLookupDependencyErrors(t *testing.T) {
	store := newFakeStore()
	service := newServiceForTest(t, store, allowAll{})
	if _, _, _, err := service.Refresh(context.Background(), RefreshInput{RefreshToken: "nonempty-refresh-token"}); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("missing session: %v", err)
	}
	user := registerAndVerify(t, service, "roles-error@example.test")
	store.rolesErr = errors.New("role storage unavailable")
	if _, _, _, err := service.Login(context.Background(), LoginInput{Email: user.Email, Password: "correct horse battery staple"}); err == nil {
		t.Fatal("login role lookup error was ignored")
	}
	store.rolesErr = nil
	credentials, _, _, err := service.Login(context.Background(), LoginInput{Email: user.Email, Password: "correct horse battery staple"})
	if err != nil {
		t.Fatal(err)
	}
	store.rolesErr = errors.New("role storage unavailable")
	if _, _, _, err := service.Refresh(context.Background(), RefreshInput{RefreshToken: credentials.RefreshToken}); err == nil {
		t.Fatal("role lookup error was ignored")
	}
}

func TestRecoveryRateLimitIsEnforced(t *testing.T) {
	store := newFakeStore()
	service := newServiceForTest(t, store, rejectLimiter{})
	if err := service.RequestPasswordReset(context.Background(), "person@example.test", "corr"); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("reset rate limit: %v", err)
	}
	if err := service.ConfirmEmail(context.Background(), "unknown-token", "corr"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("unknown verification: %v", err)
	}
}
