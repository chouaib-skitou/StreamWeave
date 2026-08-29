package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	identitycrypto "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/adapters/outbound/crypto"
	identityapp "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/application/identity"
	domain "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/domain/identity"
	"github.com/google/uuid"
)

type claimsContextKey struct{}

func (s *Server) registerIdentityRoutes() {
	if s.identity == nil || s.signer == nil {
		return
	}
	mux := s.mux
	mux.HandleFunc("POST /.well-known/register", s.register)
	mux.HandleFunc("GET /.well-known/jwks.json", s.jwks)
	if s.testMailer && s.mailbox != nil {
		mux.HandleFunc("GET /_test/mailbox/latest", s.latestTestMail)
	}
	mux.HandleFunc("POST /v1/auth/login", s.login)
	mux.HandleFunc("POST /v1/auth/refresh", s.refresh)
	mux.HandleFunc("POST /v1/auth/password-reset/request", s.resetRequest)
	mux.HandleFunc("POST /v1/auth/password-reset/confirm", s.resetConfirm)
	mux.HandleFunc("POST /v1/auth/email-verification/confirm", s.verifyEmail)
	mux.HandleFunc("POST /v1/auth/service-token", s.serviceToken)
	mux.Handle("POST /v1/auth/logout", s.auth(s.logout))
	mux.Handle("POST /v1/auth/logout-all", s.auth(s.logoutAll))
	mux.Handle("GET /v1/auth/sessions", s.auth(s.sessions))
	mux.Handle("GET /v1/users/{user_id}", s.auth(s.getUser))
	mux.Handle("POST /v1/users/{user_id}/roles", s.auth(s.assignRole))
	mux.Handle("DELETE /v1/users/{user_id}/roles/{role}", s.auth(s.removeRole))
	mux.Handle("PATCH /v1/users/{user_id}/status", s.auth(s.changeStatus))
}

func (s *Server) auth(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.signer == nil {
			problem(w, http.StatusServiceUnavailable, "service_unavailable", "Identity signing is unavailable")
			return
		}
		raw := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if raw == "" {
			problem(w, http.StatusUnauthorized, "unauthorized", "Authentication failed")
			return
		}
		audience := s.humanAudience
		if audience == "" {
			audience = "platform-api"
		}
		claims, err := s.signer.Verify(raw, audience)
		if err != nil || claims.TokenKind != "human" {
			problem(w, http.StatusUnauthorized, "unauthorized", "Authentication failed")
			return
		}
		if s.revocations != nil {
			revoked, checkErr := s.revocations.IsRevoked(r.Context(), claims.ID)
			if checkErr != nil || revoked {
				problem(w, http.StatusUnauthorized, "unauthorized", "Authentication failed")
				return
			}
		}
		next(w, r.WithContext(context.WithValue(r.Context(), claimsContextKey{}, claims)))
	})
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if !s.demoRegistration {
		problem(w, http.StatusNotFound, "not_found", "Resource not found")
		return
	}
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	user, err := s.identity.Register(r.Context(), identityapp.RegisterInput{Email: input.Email, Password: input.Password})
	if err != nil {
		writeIdentityError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, userResponse{ID: userRef(user.ID), Email: user.Email, Status: string(user.Status), EmailVerifiedAt: user.EmailVerifiedAt, CreatedAt: user.CreatedAt, Roles: []string{"customer"}})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	credentials, user, roles, err := s.identity.Login(r.Context(), identityapp.LoginInput{Email: input.Email, Password: input.Password, CorrelationID: r.Header.Get("X-Correlation-ID")})
	if err != nil {
		writeIdentityError(w, err)
		return
	}
	_ = user
	_ = roles
	writeJSON(w, http.StatusOK, tokenResponse{AccessToken: credentials.AccessToken, RefreshToken: credentials.RefreshToken, TokenType: "Bearer", ExpiresIn: credentials.AccessTokenExpiresIn, RefreshExpiresIn: credentials.RefreshExpiresIn})
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	credentials, _, _, err := s.identity.Refresh(r.Context(), identityapp.RefreshInput{RefreshToken: input.RefreshToken, CorrelationID: r.Header.Get("X-Correlation-ID")})
	if err != nil {
		writeIdentityError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tokenResponse{AccessToken: credentials.AccessToken, RefreshToken: credentials.RefreshToken, TokenType: "Bearer", ExpiresIn: credentials.AccessTokenExpiresIn, RefreshExpiresIn: credentials.RefreshExpiresIn})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	claims := requestClaims(r)
	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		problem(w, http.StatusUnauthorized, "unauthorized", "Authentication failed")
		return
	}
	userID, err := parseUserRef(claims.Subject)
	if err != nil {
		problem(w, http.StatusUnauthorized, "unauthorized", "Authentication failed")
		return
	}
	if err := s.identity.Logout(r.Context(), userID, sessionID, r.Header.Get("X-Correlation-ID")); err != nil {
		writeIdentityError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) logoutAll(w http.ResponseWriter, r *http.Request) {
	userID, err := parseUserRef(requestClaims(r).Subject)
	if err != nil {
		problem(w, 401, "unauthorized", "Authentication failed")
		return
	}
	if err := s.identity.LogoutAll(r.Context(), userID, r.Header.Get("X-Correlation-ID")); err != nil {
		writeIdentityError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (s *Server) sessions(w http.ResponseWriter, r *http.Request) {
	userID, err := parseUserRef(requestClaims(r).Subject)
	if err != nil {
		problem(w, 401, "unauthorized", "Authentication failed")
		return
	}
	sessions, err := s.identity.Sessions(r.Context(), userID)
	if err != nil {
		writeIdentityError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"sessions": sessions})
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request) {
	target, err := parseUserRef(r.PathValue("user_id"))
	if err != nil {
		problem(w, 400, "bad_request", "Invalid user identifier")
		return
	}
	claims := requestClaims(r)
	if claims.Subject != userRef(target) && !hasScope(claims, "identity:users:read") {
		problem(w, 403, "forbidden", "Permission denied")
		return
	}
	user, roles, err := s.identity.User(r.Context(), target)
	if err != nil {
		writeIdentityError(w, err)
		return
	}
	roleNames := make([]string, 0)
	seen := map[string]bool{}
	for _, role := range roles {
		if !seen[role.Role] {
			roleNames = append(roleNames, role.Role)
			seen[role.Role] = true
		}
	}
	writeJSON(w, 200, userResponse{ID: userRef(user.ID), Email: user.Email, Status: string(user.Status), EmailVerifiedAt: user.EmailVerifiedAt, CreatedAt: user.CreatedAt, Roles: roleNames})
}
func (s *Server) assignRole(w http.ResponseWriter, r *http.Request) {
	if !hasScope(requestClaims(r), "identity:roles:manage") {
		problem(w, 403, "forbidden", "Permission denied")
		return
	}
	target, err := parseUserRef(r.PathValue("user_id"))
	if err != nil {
		problem(w, 400, "bad_request", "Invalid user identifier")
		return
	}
	var input struct {
		Role string `json:"role"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	actor, _ := parseUserRef(requestClaims(r).Subject)
	if err := s.identity.AssignRole(r.Context(), target, actor, input.Role, r.Header.Get("X-Correlation-ID")); err != nil {
		writeIdentityError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) removeRole(w http.ResponseWriter, r *http.Request) {
	if !hasScope(requestClaims(r), "identity:roles:manage") {
		problem(w, 403, "forbidden", "Permission denied")
		return
	}
	target, err := parseUserRef(r.PathValue("user_id"))
	if err != nil {
		problem(w, 400, "bad_request", "Invalid user identifier")
		return
	}
	actor, _ := parseUserRef(requestClaims(r).Subject)
	if err := s.identity.RemoveRole(r.Context(), target, actor, r.PathValue("role"), r.Header.Get("X-Correlation-ID")); err != nil {
		writeIdentityError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) changeStatus(w http.ResponseWriter, r *http.Request) {
	if !hasScope(requestClaims(r), "identity:roles:manage") {
		problem(w, 403, "forbidden", "Permission denied")
		return
	}
	target, err := parseUserRef(r.PathValue("user_id"))
	if err != nil {
		problem(w, 400, "bad_request", "Invalid user identifier")
		return
	}
	var input struct {
		Status domain.UserStatus `json:"status"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	actor, _ := parseUserRef(requestClaims(r).Subject)
	if err := s.identity.ChangeUserStatus(r.Context(), target, input.Status, actor, r.Header.Get("X-Correlation-ID")); err != nil {
		writeIdentityError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (s *Server) resetRequest(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.identity.RequestPasswordReset(r.Context(), input.Email, r.Header.Get("X-Correlation-ID")); err != nil {
		writeIdentityError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
func (s *Server) resetConfirm(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.identity.ConfirmPasswordReset(r.Context(), input.Token, input.NewPassword, r.Header.Get("X-Correlation-ID")); err != nil {
		writeIdentityError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token string `json:"token"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.identity.ConfirmEmail(r.Context(), input.Token, r.Header.Get("X-Correlation-ID")); err != nil {
		writeIdentityError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) serviceToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ClientID     string   `json:"client_id"`
		ClientSecret string   `json:"client_secret"`
		Scopes       []string `json:"scopes"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.identity.ServiceToken(r.Context(), input.ClientID, input.ClientSecret, input.Scopes, r.Header.Get("X-Correlation-ID"))
	if err != nil {
		writeIdentityError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"access_token": result.AccessToken, "token_type": "Bearer", "expires_in": result.ExpiresIn})
}
func (s *Server) jwks(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, s.signer.JWKS()) }

func (s *Server) latestTestMail(w http.ResponseWriter, r *http.Request) {
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	recipient := strings.TrimSpace(r.URL.Query().Get("recipient"))
	if kind == "" || recipient == "" {
		problem(w, http.StatusBadRequest, "bad_request", "kind and recipient are required")
		return
	}
	token, err := s.mailbox.Latest(r.Context(), kind, recipient)
	if err != nil {
		problem(w, http.StatusNotFound, "not_found", "Mail not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"kind": kind, "recipient": recipient, "token": token})
}

type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
}
type userResponse struct {
	ID              string     `json:"id"`
	Email           string     `json:"email"`
	Status          string     `json:"status"`
	Roles           []string   `json:"roles"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

func requestClaims(r *http.Request) *identitycrypto.Claims {
	claims, _ := r.Context().Value(claimsContextKey{}).(*identitycrypto.Claims)
	return claims
}
func hasScope(claims *identitycrypto.Claims, wanted string) bool {
	if claims == nil {
		return false
	}
	for _, scope := range claims.Scopes {
		if scope == wanted {
			return true
		}
	}
	return false
}
func parseUserRef(value string) (uuid.UUID, error) {
	if !strings.HasPrefix(value, "usr_") {
		return uuid.Nil, errors.New("invalid user reference")
	}
	return uuid.Parse(strings.TrimPrefix(value, "usr_"))
}
func userRef(id uuid.UUID) string { return "usr_" + strings.ReplaceAll(id.String(), "-", "") }
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		problem(w, 400, "bad_request", "Invalid request body")
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		problem(w, 400, "bad_request", "Invalid request body")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func problem(w http.ResponseWriter, status int, title, detail string) {
	writeJSON(w, status, map[string]any{"type": "about:blank", "title": title, "status": status, "detail": detail})
}
func writeIdentityError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		problem(w, 401, "unauthorized", "Authentication failed")
	case errors.Is(err, domain.ErrForbidden):
		problem(w, 403, "forbidden", "Permission denied")
	case errors.Is(err, domain.ErrNotFound):
		problem(w, 404, "not_found", "Resource not found")
	case errors.Is(err, domain.ErrRateLimited):
		problem(w, 429, "too_many_requests", "Request rate limit exceeded")
	case errors.Is(err, domain.ErrConflict):
		problem(w, 409, "conflict", "Request conflicts with existing state")
	case errors.Is(err, domain.ErrDependency):
		problem(w, 503, "service_unavailable", "Required dependency unavailable")
	case errors.Is(err, domain.ErrAlreadyUsed), errors.Is(err, domain.ErrExpired):
		problem(w, 401, "unauthorized", "Authentication failed")
	default:
		problem(w, 500, "internal_error", "Internal server error")
	}
}
