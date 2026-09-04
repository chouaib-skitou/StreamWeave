---
name: Gateway Service
type: architecture-spine
purpose: build-substrate
altitude: feature
paradigm: Clean / Hexagonal Architecture
scope: Public HTTP edge, authentication verification, coarse authorization, routing, limits, errors, and telemetry
status: ready-for-implementation
created: 2026-09-04
updated: 2026-09-04
sources:
  - 01_event_driven_ecommerce_order_management_platform.md
  - docs/brief.md
  - docs/project_context.md
  - docs/prd/PRD-002-gateway-service.md
  - docs/architecture/identity-service.md
companions:
  - contracts/openapi/gateway.openapi.yaml
  - docs/design/gateway-route-matrix.md
---

# Gateway Service Architecture

## Design paradigm

The Gateway is an independently deployable Go application with transport, application, domain-policy, and infrastructure adapter boundaries:

```text
HTTP adapter
    ↓
Request pipeline and route policy
    ↓
Application ports: verifier, limiter, router, upstream client, telemetry
    ↓
Infrastructure adapters: JWKS/crypto, Redis, HTTP clients, OTel/Prometheus
```

The domain and policy code must not depend on chi, PostgreSQL, Redis clients, JWT libraries, Kubernetes, or OpenTelemetry. HTTP and infrastructure concerns remain adapters.

## Boundary and ownership

| Concern | Gateway | Owning component |
|---|---|---|
| Public HTTP origin and route allowlist | Owns | Gateway |
| Request/trace IDs and error envelope | Owns | Gateway |
| JWT signature and claim verification | Verifies | Identity issues; services may independently verify |
| Coarse scope gate | Owns | Gateway |
| Resource ownership and domain authorization | Must not decide | Orders or Identity use case |
| Users, sessions, credentials, orders | Must not store | Identity or Orders PostgreSQL |
| Rate-limit counters | Ephemeral adapter | Redis |
| Order idempotency | Must preserve header | Orders |
| Service-to-service credentials | Uses dedicated principal | Identity |

## Request pipeline

1. Assign or validate request and correlation IDs.
2. Start or continue the W3C trace.
3. Match method and path against the explicit route registry.
4. Apply request size and content-type limits.
5. Apply route-aware rate limiting.
6. Verify the Bearer token when required, including issuer, audience, algorithm, time claims, `kid`, and required scope class.
7. Remove all client-supplied internal identity headers.
8. Attach safe authenticated context and Gateway service credentials to the internal call.
9. Enforce the route timeout and constrained retry policy.
10. Normalize the response, error envelope, response headers, and telemetry.

## Trust model

Identity is the only component that owns private signing keys. Gateway trusts only the configured Identity issuer, audience, and EdDSA algorithm. JWKS is cached in memory with bounded freshness; an unknown key ID permits one refresh, then fails closed. A stale cache is not a reason to accept a token whose key cannot be verified.

Downstream services must authenticate the Gateway service principal before trusting Gateway-provided actor context. The context contains only the actor subject, actor type, active actor session ID, scopes, request ID, correlation ID, and trace context. It does not contain the original bearer token or credentials.

## Resilience policy

- No dependency is contacted by liveness.
- Readiness validates configuration and the dependencies required to enforce the configured security policy.
- Authentication and mutation routes fail closed when the verifier or required limiter cannot operate.
- Safe reads return bounded `503`/`504` errors when dependencies are unavailable; they never expose upstream topology.
- `POST` requests, login, refresh, logout, and password operations are never automatically retried.
- A single pre-response retry is allowed only for an explicitly idempotent read and only for transport-level connection failure.

## Data and state

The Gateway has no durable database. In-memory JWKS and configuration caches are disposable. Redis keys are namespaced, TTL-bound, and operationally disposable. Orders owns `Idempotency-Key` semantics and durable response replay.

## Deployment shape

The service runs as a non-root container behind an ingress/load balancer and is exposed internally as a Kubernetes `ClusterIP`. It uses a ServiceAccount, ConfigMap, Secret references, liveness/readiness/startup probes as appropriate, resource requests/limits, PDB, HPA, and a restrictive NetworkPolicy. It has no database egress. The complete environment contract is in `docs/design/gateway-deployment.md`.

## Architecture invariants

- **G-AI-1:** No open proxy; every route is explicitly registered.
- **G-AI-2:** No business rule or durable business state in Gateway.
- **G-AI-3:** Invalid or unverifiable authentication fails closed.
- **G-AI-4:** Client-controlled identity headers are never trusted.
- **G-AI-5:** Redis failure never turns a protected route into an unbounded bypass.
- **G-AI-6:** Every response is correlated and every failure is safe to expose.
