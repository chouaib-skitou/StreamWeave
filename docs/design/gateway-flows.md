# Gateway interaction flows

## Anonymous Identity operation

1. Client sends an explicitly documented authentication request.
2. Gateway assigns request and trace identity, validates body and content type, and evaluates the anonymous-auth limit.
3. Gateway routes only to Identity, without attempting JWT verification for an anonymous operation.
4. Gateway returns Identity's documented result in the common response envelope and never logs credentials.

## Protected order read

1. Client sends a Bearer access token and optional cursor/filter.
2. Gateway evaluates the authenticated-read limit and verifies the token against the cached Identity JWKS.
3. Gateway checks the route's coarse scope class and strips spoofable internal headers.
4. Gateway calls Orders with the Gateway service credential and safe actor context.
5. Orders performs ownership/resource authorization and returns the order view or a safe error.

## Protected order command

1. Client sends a Bearer token, a bounded JSON body, and an `Idempotency-Key`.
2. Gateway evaluates the order-write limit, verifies the token, and checks the coarse write scope.
3. Gateway forwards the idempotency key unchanged to Orders exactly once; it never retries this request.
4. Orders owns durable idempotency, business validation, ownership, and asynchronous workflow initiation.
5. Gateway returns `202 Accepted` without claiming that downstream processing is complete.

## Dependency failure

1. Gateway records a low-cardinality failure outcome with request and trace correlation.
2. If verification or a required limiter cannot operate, protected traffic fails closed.
3. If an upstream call times out, Gateway returns bounded `503` or `504` Problem Details.
4. The client can retry only according to the public contract; Gateway does not replay ambiguous mutations.
