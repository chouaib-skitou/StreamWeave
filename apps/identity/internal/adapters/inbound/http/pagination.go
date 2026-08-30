package httpadapter

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	identityapp "github.com/chouaib-skitou/streamweave/apps/identity/internal/application/identity"
	"github.com/google/uuid"
)

const (
	defaultPageSize = 25
	maxPageSize     = 100
)

var (
	allowedUserStatuses = map[string]bool{"ACTIVE": true, "DISABLED": true, "PENDING_VERIFICATION": true}
	allowedUserRoles    = map[string]bool{"admin": true, "customer": true, "support": true, "finance": true}
)

type pageQuery struct {
	limit         int32
	status        string
	role          string
	emailPrefix   string
	cursorCreated *time.Time
	cursorID      uuid.UUID
}

type cursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        uuid.UUID `json:"id"`
}

func parsePageQuery(request *http.Request) (pageQuery, error) {
	values := request.URL.Query()
	rawPrefix := values.Get("email_prefix")
	if strings.ContainsAny(rawPrefix, "\r\n") {
		return pageQuery{}, errors.New("invalid email prefix")
	}
	escapedPrefix := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(strings.TrimSpace(rawPrefix)))
	result := pageQuery{limit: defaultPageSize, status: strings.TrimSpace(values.Get("status")), role: strings.TrimSpace(values.Get("role")), emailPrefix: escapedPrefix}
	if raw := strings.TrimSpace(values.Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > maxPageSize {
			return pageQuery{}, errors.New("limit must be between 1 and 100")
		}
		result.limit = int32(value)
	}
	if len(result.emailPrefix) > 320 || strings.ContainsAny(result.emailPrefix, "\r\n") {
		return pageQuery{}, errors.New("invalid email prefix")
	}
	if raw := strings.TrimSpace(values.Get("cursor")); raw != "" {
		createdAt, id, err := decodeCursor(raw)
		if err != nil {
			return pageQuery{}, errors.New("invalid cursor")
		}
		result.cursorCreated, result.cursorID = &createdAt, id
	}
	return result, nil
}

func (q pageQuery) userQuery() identityapp.UserPageQuery {
	return identityapp.UserPageQuery{Status: q.status, Role: q.role, EmailPrefix: q.emailPrefix, CursorCreatedAt: q.cursorCreated, CursorID: q.cursorID, Limit: q.limit}
}

func (q pageQuery) sessionQuery() identityapp.SessionPageQuery {
	return identityapp.SessionPageQuery{CursorCreatedAt: q.cursorCreated, CursorID: q.cursorID, Limit: q.limit}
}

func encodeCursor(createdAt time.Time, id uuid.UUID) string {
	payload, _ := json.Marshal(cursor{CreatedAt: createdAt.UTC(), ID: id})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeCursor(value string) (time.Time, uuid.UUID, error) {
	if len(value) > 256 {
		return time.Time{}, uuid.Nil, errors.New("cursor too long")
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	var valueCursor cursor
	if err := json.Unmarshal(payload, &valueCursor); err != nil || valueCursor.ID == uuid.Nil || valueCursor.CreatedAt.IsZero() {
		return time.Time{}, uuid.Nil, errors.New("invalid cursor")
	}
	return valueCursor.CreatedAt.UTC(), valueCursor.ID, nil
}
