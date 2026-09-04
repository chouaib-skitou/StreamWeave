package gateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	domain "github.com/chouaib-skitou/streamweave/apps/gateway/internal/domain/gateway"
)

type Service struct {
	verifier ClaimsVerifier
	limiter  RateLimiter
	tokens   ServiceTokenProvider
	upstream UpstreamClient
	metrics  domain.Metrics
}

type ClaimsVerifier = domain.ClaimsVerifier
type RateLimiter = domain.RateLimiter
type ServiceTokenProvider = domain.ServiceTokenProvider
type RefreshableServiceTokenProvider = domain.RefreshableServiceTokenProvider
type UpstreamClient = domain.UpstreamClient
type Response = domain.Response
type Actor = domain.Actor
type RoutePolicy = domain.RoutePolicy

func NewService(verifier ClaimsVerifier, limiter RateLimiter, tokens ServiceTokenProvider, upstream UpstreamClient) (*Service, error) {
	if verifier == nil || limiter == nil || tokens == nil || upstream == nil {
		return nil, errors.New("gateway dependencies are required")
	}
	return &Service{verifier: verifier, limiter: limiter, tokens: tokens, upstream: upstream}, nil
}

func (s *Service) SetMetrics(metrics domain.Metrics) { s.metrics = metrics }

func (s *Service) Authorize(ctx context.Context, request *http.Request, policy RoutePolicy, source string) (Actor, string, error) {
	if request == nil {
		return Actor{}, "", domain.ErrBadRequest
	}
	var actor Actor
	if policy.Auth == domain.AuthNone {
		actor = Actor{}
	} else {
		if len(request.Header.Values("Authorization")) != 1 {
			s.observeAuthFailure(policy.Pattern, "unauthorized")
			return Actor{}, "", domain.ErrUnauthorized
		}
		raw, ok := bearerToken(request.Header.Get("Authorization"))
		if !ok {
			s.observeAuthFailure(policy.Pattern, "unauthorized")
			return Actor{}, "", domain.ErrUnauthorized
		}
		verified, err := s.verifier.Verify(ctx, raw)
		if err != nil {
			s.observeAuthFailure(policy.Pattern, "unauthorized")
			return Actor{}, "", domain.ErrUnauthorized
		}
		actor = verified.Normalized()
		if actor.Subject == "" || actor.Type != "human" || !policy.Allows(actor.Scopes) {
			s.observeAuthFailure(policy.Pattern, "forbidden")
			return Actor{}, "", domain.ErrForbidden
		}
		if allowed, err := s.limiter.Allow(ctx, limitKeyWithActor(policy, source, actor), policy.LimitKeyClass()); err != nil {
			s.observeLimiterError(policy.LimitKeyClass())
			return Actor{}, "", domain.ErrDependency
		} else if !allowed {
			s.observeRateLimit(policy.Pattern, policy.LimitKeyClass())
			return Actor{}, "", domain.ErrRateLimited
		}
		return actor, raw, nil
	}
	if allowed, err := s.limiter.Allow(ctx, limitKey(request, policy, source), policy.LimitKeyClass()); err != nil {
		s.observeLimiterError(policy.LimitKeyClass())
		return Actor{}, "", domain.ErrDependency
	} else if !allowed {
		s.observeRateLimit(policy.Pattern, policy.LimitKeyClass())
		return Actor{}, "", domain.ErrRateLimited
	}
	return actor, "", nil
}

func (s *Service) Forward(ctx context.Context, owner string, actor *Actor, request domain.ForwardRequest, policy RoutePolicy) (*Response, error) {
	token, err := s.tokens.Token(ctx)
	if err != nil {
		return nil, domain.ErrDependency
	}
	response, err := s.forwardUpstream(ctx, owner, token, actor, request)
	if err != nil {
		return nil, err
	}
	if response == nil || response.StatusCode < 100 || response.StatusCode > 599 {
		return nil, domain.ErrUpstreamResponse
	}
	if response.StatusCode == http.StatusUnauthorized && policy.RetrySafeRead {
		if refresher, ok := s.tokens.(RefreshableServiceTokenProvider); ok {
			freshToken, refreshErr := refresher.Refresh(ctx)
			if refreshErr != nil {
				return nil, domain.ErrDependency
			}
			response, err = s.forwardUpstream(ctx, owner, freshToken, actor, request)
			if err != nil {
				return nil, err
			}
			if response == nil || response.StatusCode < 100 || response.StatusCode > 599 {
				return nil, domain.ErrUpstreamResponse
			}
		}
	}
	return response, nil
}

func (s *Service) forwardUpstream(ctx context.Context, owner, token string, actor *Actor, request domain.ForwardRequest) (*Response, error) {
	started := time.Now()
	response, err := s.upstream.Forward(ctx, owner, token, actor, request)
	route := request.Route
	if route == "" {
		route = "unknown"
	}
	outcome := "success"
	if err != nil {
		outcome = "error"
	}
	if response != nil && response.StatusCode >= 400 {
		outcome = "http_" + fmt.Sprint(response.StatusCode)
	}
	if s.metrics != nil {
		s.metrics.ObserveUpstreamRequest(owner, route, outcome)
		s.metrics.ObserveUpstreamDuration(owner, route, time.Since(started))
	}
	return response, err
}

func (s *Service) observeAuthFailure(route, reason string) {
	if s.metrics != nil {
		s.metrics.ObserveAuthFailure(route, reason)
	}
}

func (s *Service) observeRateLimit(route, class string) {
	if s.metrics != nil {
		s.metrics.ObserveRateLimitRejection(route, class)
	}
}

func (s *Service) observeLimiterError(class string) {
	if s.metrics != nil {
		s.metrics.ObserveRateLimiterError(class)
	}
}

func limitKey(request *http.Request, policy RoutePolicy, source string) string {
	return fmt.Sprintf("%s|anonymous|%s|%s", policy.LimitKeyClass(), source, request.Method)
}

func limitKeyWithActor(policy RoutePolicy, source string, actor Actor) string {
	return fmt.Sprintf("%s|%s|%s|%s", policy.LimitKeyClass(), actor.Subject, source, policy.Method)
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}
