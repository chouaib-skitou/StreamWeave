package identityclient

import (
	"context"
	"errors"
	"strings"
)

type StaticToken string

func (s StaticToken) Token(context.Context) (string, error) {
	if strings.TrimSpace(string(s)) == "" {
		return "", errors.New("static service token is empty")
	}
	return string(s), nil
}

func (s StaticToken) Refresh(ctx context.Context) (string, error) { return s.Token(ctx) }
