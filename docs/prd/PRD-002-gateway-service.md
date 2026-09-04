---
title: Gateway Service
status: ready-for-implementation
created: 2026-09-04
updated: 2026-09-04
---

# PRD: Gateway Service

## 0. Document purpose

This document is the product and behavior contract for the public Gateway of StreamWeave. It is derived from `01_event_driven_ecommerce_order_management_platform.md`, `docs/brief.md`, `docs/project_context.md`, and the approved Identity documentation. It must be accepted before Gateway implementation begins.

The Gateway is a lightweight public edge, not a business-service replacement or a general API-management product. It authenticates requests, applies coarse policy, routes them to the owning service, and presents a stable public HTTP contract.

## 1. Vision

Give clients one reliable, observable, secure HTTP entry point for StreamWeave. A client should not need to know which service owns orders or identity, while each owning service remains authoritative for its own data and business decisions.

## 2. Outcomes and non-goals

### Desired outcomes

- One public origin for versioned REST APIs.
- Consistent request IDs, tracing, errors, authentication failures, rate limits, and access logs.
- Early rejection of invalid or insufficiently scoped requests without duplicating business authorization.
- Honest asynchronous order semantics: order creation returns `202 Accepted`.
- Safe operation when Identity, Redis, or an upstream service is degraded.

### Non-goals

- No users, credentials, sessions, orders, or other durable Gateway-owned state.
- No SQL, direct database access, business workflows, inventory checks, payment logic, or resource-ownership decisions.
- No public service-token issuance, Kafka consumer, or human-token forwarding through events.
- No wildcard CORS, unrestricted proxying, request-body logging, or generic open proxy behavior.

## 3. Actors and journeys

- **Customer:** authenticates through the public edge and creates or observes their own orders.
- **Operator:** uses a valid role-derived scope; the owning service performs final resource authorization.
- **Service client:** calls an explicitly documented route with a machine token when allowed.
- **Platform operator:** observes health, metrics, traces, rate-limit behavior, and upstream failures without accessing customer secrets.

### Primary journeys

1. A client calls an Identity route; the Gateway validates the request and routes it without exposing internal hostnames.
2. A client calls `POST /v1/orders` with a valid access token and `Idempotency-Key`; the Gateway validates coarse scope and forwards the request to Orders, which owns idempotency and returns `202`.
3. A request with an expired token, unknown signing key, missing scope, exhausted limit, or unavailable dependency receives a documented RFC 9457 Problem Details response.

## 4. Product requirements

### FR-1 — Public routing

The Gateway shall expose only an explicit allowlist of versioned routes, plus the Identity development-only `/.well-known/register` route when that feature is enabled locally. It shall route Identity authentication/session paths to Identity and order paths to Orders. Unknown paths and unsupported methods shall not be proxied.

### FR-2 — Request identity and tracing

The Gateway shall accept a valid `X-Request-ID`, generate one when absent, propagate W3C `traceparent`, and return the request ID in every response. Client-supplied correlation headers shall be normalized before internal propagation.

### FR-3 — JWT verification

Protected routes shall require a Bearer access token. The Gateway shall verify EdDSA signatures using Identity JWKS and validate algorithm, issuer, audience, time claims, `sub`, `jti`, and `kid`. An unknown `kid` may trigger one controlled JWKS refresh; verification then fails closed.

### FR-4 — Coarse authorization

Each route shall declare a required scope class. Gateway authorization is an early boundary check only. Orders and Identity remain responsible for resource ownership and domain authorization.

### FR-5 — Safe context propagation

The Gateway shall remove spoofable identity headers from the incoming request. It shall call internal services using an authenticated Gateway service principal and forward only a documented safe actor context, request/correlation IDs, and trace context. Human bearer tokens shall not be forwarded to downstream services or emitted in logs/events.

### FR-6 — Rate limiting

The Gateway shall apply route-aware Redis-backed limits to anonymous and authenticated traffic. A rejected request receives `429 Too Many Requests`, RFC 9457 details, and `Retry-After`. Redis is disposable but a configured security limit must never be silently bypassed when its enforcement dependency fails.

### FR-7 — Resilience and error mapping

The Gateway shall enforce bounded route-specific upstream timeouts. It shall not retry authentication or non-idempotent mutations. It may make one constrained retry only for a safe idempotent read before any upstream response is received. Internal errors shall be mapped to stable Problem Details without leaking topology or credentials.

### FR-8 — Operational visibility

The Gateway shall emit low-cardinality metrics, structured redacted logs, and OpenTelemetry traces. Every request shall be attributable by templated route, status, request ID, and trace ID without using user IDs, raw URLs, tokens, or payloads as unbounded labels.

### FR-9 — Health semantics

`/health/live` shall report process liveness without requiring dependencies. `/health/ready` shall report whether the Gateway can safely accept traffic under its configured policy; Orders availability is not required for liveness and is surfaced as a request-level upstream failure.

## 5. Success criteria

- Every documented route has an explicit owner, scope class, timeout, rate-limit class, and failure mapping.
- Invalid, expired, wrongly-audience, wrongly-signed, or insufficiently scoped tokens are rejected consistently.
- Orders receives no client-controlled identity context and retains final authorization and idempotency ownership.
- No secret, bearer token, password, or full request body appears in logs, metrics, traces, or committed configuration.
- The service can be built, tested, containerized, observed, and deployed with the operational contracts in this document.
- The implementation satisfies the repository quality gate: formatting, static analysis, race-enabled tests, security tests, and at least 85% application coverage.

## 5.1 Contract impact

- **API:** `contracts/openapi/gateway.openapi.yaml` defines the versioned facade, headers, status codes, security schemes, pagination, and Problem Details.
- **Events:** none; Gateway does not own or publish business events.
- **Data:** none; Gateway has no durable database. Redis contains only TTL-bound rate-limit state.
- **Operations:** deployment, telemetry, alert vocabulary, and recovery behavior are defined in the Gateway design and runbooks.

## 6. Release and compatibility

The Gateway public API is versioned under `/v1`. Additive response fields and new optional query parameters are backward-compatible. Removing or changing the meaning of a route, scope, status code, error shape, required header, or security behavior is breaking and requires a major API/version decision under the repository release policy.

## 7. Deferred decisions

- Frontend-specific CORS origins remain disabled until an approved client origin exists.
- WebSockets, GraphQL, file uploads, arbitrary proxying, and API aggregation are out of scope.
- Full circuit-breaker policy and adaptive global traffic management are deferred until production traffic evidence requires them.
