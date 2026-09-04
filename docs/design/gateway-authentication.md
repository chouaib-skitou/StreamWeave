# Gateway authentication and authorization design

## Token validation

For every protected route, Gateway:

1. Requires exactly one Bearer credential in `Authorization`.
2. Parses without accepting a token-supplied algorithm.
3. Allows only `EdDSA`/Ed25519 as configured by the Identity contract.
4. Loads the public key from the configured Identity JWKS and matches `kid`.
5. Refreshes JWKS once when the key ID is unknown, with bounded timeout and no request fan-out.
6. Validates issuer, audience, signature, `exp`, `nbf`, `iat`, `sub`, and `jti`.
7. Applies the route's required coarse scope class.
8. Emits only a reason category such as `missing`, `malformed`, `expired`, `invalid_signature`, `wrong_audience`, or `insufficient_scope`.

Unknown keys, invalid signatures, invalid claims, and unavailable verification dependencies fail closed. No token value is placed in logs, metrics, traces, responses, or error details.

## Context trust boundary

The incoming request is untrusted. Before proxying, Gateway strips client-provided values for:

`X-StreamWeave-Actor-ID`, `X-StreamWeave-Actor-Type`, `X-StreamWeave-Scopes`, `X-Request-ID`, `X-Correlation-ID`, and any internal service-authentication header.

Gateway then creates a trusted internal context after authenticating its service principal. The context is signed/authenticated by the internal call, contains the actor subject/type and normalized scopes, and is used for attribution only. Orders performs the final ownership and state checks.

## Authentication route policy

Login, refresh, password reset, and email verification are anonymous but abuse-sensitive. They have separate rate-limit classes, do not retry automatically, do not reveal account existence where the Identity contract forbids it, and return the Identity-owned public error semantics through the common Problem Details envelope.

## Machine-to-machine policy

Gateway uses the dedicated `svc_gateway` principal to call internal services. The service token has a separate machine audience, a short TTL, and a scope subset explicitly granted by Identity. Human access tokens are never reused as service credentials. Key and credential rotation is documented in the Identity and Gateway runbooks.

## CORS and browser policy

The default deployment has no wildcard CORS. An approved client-origin allowlist may be configured per environment. Credentials, methods, headers, and max age are explicit. CORS is not an authorization mechanism.
