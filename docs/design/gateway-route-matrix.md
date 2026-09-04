# Gateway route and policy matrix

This matrix is the reviewable source for Gateway route registration. The OpenAPI contract in `contracts/openapi/gateway.openapi.yaml` is the client-facing source of truth; this document adds ownership, security, resilience, and operational policy.

| Method | Public path | Owner | Auth | Scope class | Limit class | Timeout | Retry |
|---|---|---|---|---|---|---|---|
| GET | `/health/live` | Gateway | None | None | health | 1s | none |
| GET | `/health/ready` | Gateway | None | None | health | 2s | none |
| GET | `/metrics` | Gateway | Internal network | ops | metrics | 2s | none |
| GET | `/v1/docs` | Gateway | None | None | health | 2s | none |
| GET | `/v1/openapi.yaml` | Gateway | None | None | health | 2s | none |
| POST | `/v1/auth/login` | Identity | None | anonymous-auth | auth-write | 5s | none |
| POST | `/v1/auth/refresh` | Identity | None | anonymous-auth | auth-write | 5s | none |
| POST | `/v1/auth/password-reset/request` | Identity | None | anonymous-auth | auth-write | 5s | none |
| POST | `/v1/auth/password-reset/confirm` | Identity | None | anonymous-auth | auth-write | 5s | none |
| POST | `/v1/auth/email-verification/confirm` | Identity | None | anonymous-auth | auth-write | 5s | none |
| POST | `/v1/auth/logout` | Identity | Bearer | session-write | auth-write | 5s | none |
| POST | `/v1/auth/logout-all` | Identity | Bearer | session-write | auth-write | 5s | none |
| GET | `/v1/auth/sessions` | Identity | Bearer | session-read | authenticated-read | 5s | one transport retry |
| POST | `/v1/orders` | Orders | Bearer | `orders:write:self` or operator scope | order-write | 10s | none |
| GET | `/v1/orders` | Orders | Bearer | `orders:read:self` or operator scope | order-read | 5s | one transport retry |
| GET | `/v1/orders/{order_id}` | Orders | Bearer | `orders:read:self` or operator scope | order-read | 5s | one transport retry |
| POST | `/v1/orders/{order_id}/cancel` | Orders | Bearer | `orders:cancel:self` or operator scope | order-write | 10s | none |

## Common policy

- Unknown paths and methods return `404` or `405` without an upstream call.
- Protected requests require a valid Bearer access token. Gateway checks coarse scope only; the owner makes the final resource decision.
- `X-Request-ID` is accepted only within the documented format and is generated otherwise. The response always includes it.
- `traceparent` is propagated according to W3C rules. `Authorization` is never logged or copied to Kafka.
- `POST /v1/orders` requires `Idempotency-Key`; the Gateway validates size/format and forwards it unchanged. Orders owns uniqueness and replay semantics.
- `POST /v1/orders` also carries the blueprint's `customer_id`; Gateway treats it as untrusted input and Orders verifies actor/customer ownership. Client input never supplies the authoritative item price.
- Request bodies have bounded size; unsupported content types are rejected before proxying.
- All non-success responses use RFC 9457 Problem Details with a stable `type`, safe `title`/`detail`, `status`, `instance`, and `request_id`.

## Scope classes

| Class | Meaning |
|---|---|
| `anonymous-auth` | Public only for the stated authentication lifecycle operation; subject to aggressive abuse controls. |
| `session-read` / `session-write` | Requires an authenticated user session; Identity remains authoritative. |
| `orders:read:self` | Allows the request to reach Orders; Orders checks ownership. |
| `orders:write:self` | Allows a customer mutation candidate; Orders validates business state and ownership. |
| operator scope | Allows an operator candidate; Orders still checks role and resource policy. |
| `ops` | Internal monitoring access only; never internet-public. |

## Explicitly not routed

- Identity `/.well-known/jwks.json` and internal service-token issuance are dependency/internal surfaces, not public Gateway APIs.
- Identity administration routes are not public until an explicit admin route policy is approved.
- Arbitrary service paths, database endpoints, debug endpoints, and internal health endpoints are never proxied.
