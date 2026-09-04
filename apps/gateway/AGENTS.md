# Gateway Service Instructions

The Gateway is the public edge, not a business service.

- Keep domain and route-policy logic independent from HTTP, chi, Redis clients, JWT libraries, Kubernetes, and OpenTelemetry.
- Register only explicit routes from `contracts/openapi/gateway.openapi.yaml`; never create an open proxy.
- Verify Identity-issued EdDSA JWTs with issuer/audience/claim/key allowlists and fail closed on verifier uncertainty.
- Strip client-controlled internal headers before adding authenticated Gateway context.
- Use a dedicated Gateway service principal for downstream calls; never forward human bearer tokens.
- Do not add PostgreSQL, SQL, repositories, durable business state, Kafka consumers, or business authorization to Gateway.
- Preserve `Idempotency-Key` for Orders; Orders owns idempotency and resource authorization.
- Keep Redis disposable and fail closed for protected rate-limit policies when enforcement is unavailable.
- Never retry authentication or non-idempotent mutations.
- Use RFC 9457 Problem Details, request/trace correlation, low-cardinality telemetry, and strict redaction.
- Never commit secrets, `.env` files, tokens, private keys, or generated artifacts.
- Update the Gateway docs and contract when behavior changes; use a conventional scoped commit and the branch/PR flow.
