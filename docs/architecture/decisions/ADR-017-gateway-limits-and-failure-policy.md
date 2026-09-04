# ADR-017: Route-aware limits and fail-closed protected traffic

- **Status:** accepted
- **Date:** 2026-09-04
- **Owners:** Platform architecture and operations
- **Scope:** Gateway rate limiting and dependency failure behavior

## Context

Authentication endpoints are abuse-sensitive and order mutations must remain bounded. Redis is explicitly disposable, so it cannot become business state. Silently bypassing a configured limit during Redis failure creates an accidental security mode; restarting the whole Gateway for every upstream outage creates a cascading failure.

## Decision

Gateway uses namespaced, TTL-bound Redis counters with route-specific policies and dimensions appropriate to anonymous or authenticated traffic. Responses include `429` and `Retry-After`. If the limiter cannot enforce a protected policy, authentication and mutation routes fail closed with a safe `503`; safe health/read behavior is determined by the route policy and is never an unbounded bypass.

Gateway uses bounded per-route upstream deadlines. It never retries authentication or non-idempotent requests. It may retry one transport-level failure for an explicitly idempotent read before receiving a response.

## Consequences

- Abuse controls are explicit and observable.
- Redis outage can reduce availability for protected traffic, but cannot silently remove a security control.
- Orders remains the owner of idempotency; the Gateway only validates and forwards `Idempotency-Key`.
- Operations can distinguish limiter, verifier, and upstream failures through low-cardinality telemetry.

## Alternatives rejected

- **Unlimited fallback when Redis is unavailable:** unsafe and non-deterministic.
- **Retry every request:** duplicates mutations and auth side effects.
- **Store limits in PostgreSQL:** adds durable state and contention to a disposable concern.

## Failure modes considered

- Redis outage cannot silently create an unlimited protected route.
- A timed-out mutation is not retried because the remote acceptance state is ambiguous.
- An exhausted limit returns a retryable public response without leaking the key dimension.

## Validation

The decision is reflected by `gateway-resilience.md`, `gateway-rate-limit-outage.md`, and STORY-017.
