package gateway

import (
	"context"
	"sort"
	"strings"
)

type Actor struct {
	Subject   string
	Type      string
	Roles     []string
	Scopes    []string
	SessionID string
}

func (a Actor) Normalized() Actor {
	a.Subject = strings.TrimSpace(a.Subject)
	a.Type = strings.TrimSpace(a.Type)
	a.Roles = uniqueSorted(a.Roles)
	a.Scopes = uniqueSorted(a.Scopes)
	return a
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

type contextKey string

const actorContextKey contextKey = "streamweave.gateway.actor"

func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorContextKey, actor.Normalized())
}
func ActorFromContext(ctx context.Context) (Actor, bool) {
	actor, ok := ctx.Value(actorContextKey).(Actor)
	return actor, ok
}
