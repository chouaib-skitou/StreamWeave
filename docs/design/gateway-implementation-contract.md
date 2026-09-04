# Gateway implementation contract

This document closes the implementation-level defaults that are intentionally not repeated in the public OpenAPI description. Environment values may tighten these limits, but production values must not weaken the stated safety bounds without an ADR.

## Port contract

| Surface | Port |
|---|---:|
| Gateway container listen address | `:8080` |
| Local host mapping | `localhost:8081 -> gateway:8080` |
| Kubernetes Gateway Service | `8080 -> targetPort 8080` |
| Existing local Identity mapping | `localhost:8080 -> identity:8080` |

The local host ports are intentionally different. Internal container and Kubernetes Service ports are independently namespaced and may both use `8080`.

## Request limits and identity

| Policy | Default | Hard upper bound |
|---|---:|---:|
| Accepted `X-Request-ID` length | 1–128 bytes | 128 bytes |
| Accepted request ID characters | `[A-Za-z0-9._:-]` | same |
| Generated request ID | ULID with `sw_` prefix | n/a |
| Correlation ID | same validation as request ID; generated from request ID when absent | 128 bytes |
| Authentication body | 1 MiB | 1 MiB |
| Order body | 2 MiB | 5 MiB |
| Other public body | 1 MiB | 5 MiB |
| Page limit | 25 | 100 |

The Gateway rejects invalid client IDs rather than copying them. It returns the effective request ID in `X-Request-ID` and includes it as `request_id` in Problem Details.

## JWKS and service credential cache

- Initial readiness requires one successful JWKS load.
- A successful JWKS set is refreshed every 5 minutes or sooner when its cache-control policy requires it.
- An unknown `kid` triggers one single-flight refresh for the request; a second miss fails closed.
- A cache older than 15 minutes is unusable for protected traffic, even if the upstream Identity endpoint is currently unavailable.
- The Gateway service credential is cached only until 30 seconds before expiry and is refreshed through the Identity internal contract.
- A downstream `401` caused by service-credential expiry permits one credential refresh and one replay only when the original operation is a safe idempotent read. Mutations are never replayed.

## Rate-limit defaults

Redis keys use the prefix `streamweave:gateway:ratelimit:v1`, contain only a one-way subject/source fingerprint, route class, and time bucket, and always have an expiry.

| Limit class | Default policy | Dimensions |
|---|---|---|
| `auth-write` | 10 requests/minute, burst 5 | source fingerprint; account fingerprint when present |
| `demo-registration` | 5 requests/minute, burst 2 | source fingerprint |
| `authenticated-read` | 120 requests/minute, burst 30 | subject + source fingerprint + route class |
| `order-read` | 120 requests/minute, burst 30 | subject + source fingerprint + route class |
| `order-write` | 30 requests/minute, burst 10 | subject + source fingerprint + route class |
| `health` | local process protection only | no Redis dependency |
| `metrics` | internal network policy | no public client policy |

Limit evaluation is atomic. A rejected request returns `429` and a bounded integer `Retry-After`. Limiter failure returns `503` for `auth-write`, `order-write`, and session mutations. `order-read` and authenticated session reads also fail closed by default; health remains local.

## Upstream and response rules

- Identity and Orders are addressed by configured logical service URLs; clients never choose an upstream.
- Internal calls use `Authorization: Bearer <short-lived svc_gateway token>` plus `X-StreamWeave-Actor-ID`, `X-StreamWeave-Actor-Type`, `X-StreamWeave-Scopes`, `X-Request-ID`, and `X-Correlation-ID`. Actor type is `human` or `service`; scopes are sorted, space-delimited values. Downstream trusts these fields only after validating the service token.
- Only documented response headers are copied. Hop-by-hop headers and internal topology headers are removed.
- A valid upstream Problem Details response is preserved after redaction of unsafe fields; an invalid or unsafe error body is replaced with the Gateway error for the mapped status.
- Upstream `2xx`, `3xx`, `4xx`, and `5xx` statuses are copied only for documented operations. Connect failures map to `503`; deadline failures map to `504`.
- Redirects are not followed by the Gateway and are not introduced for API routes.
- Response buffering is bounded; oversized upstream responses are terminated with `502`.

## Readiness and shutdown

Readiness is `ready` only when configuration validates, the first JWKS load succeeded, the limiter can enforce all required default policies, and the Gateway is accepting traffic. Identity and Orders availability are not required for liveness. On shutdown, readiness turns false before in-flight requests drain within the configured grace period.

## Security and configuration invariants

- Metrics are internal-only (`x-internal: true`), bound to an internal interface or protected by network policy, and never exposed through the public ingress.
- TLS is mandatory for staging/production Redis and internal HTTP where the environment supports TLS; insecure local mode is explicit and local-only.
- CORS defaults to an empty allowlist. `*` is invalid when credentials are enabled.
- Secrets are supplied by secret references/environment injection and are never printed at startup.
