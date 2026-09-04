# Gateway resilience and failure design

## Timeout budgets

Timeouts are configured per route class and include connection, header, body, and total request budgets. The proxy stops reading and closes the upstream exchange when the budget is exceeded. Error responses do not disclose which internal host timed out.

| Class | Total budget | Intended use |
|---|---:|---|
| health | 1–2s | Gateway process and readiness checks |
| auth | 5s | Identity authentication/session operations |
| order-read | 5s | Order query operations |
| order-write | 10s | Order command candidates |

Exact values are environment configuration with safe production bounds; they are not client-controlled.

## Retry rules

- Never retry login, refresh, logout, password, verification, order creation, or cancellation.
- Allow at most one retry for a safe GET/HEAD when failure occurs before upstream headers and the route policy is explicitly idempotent.
- Never retry on `4xx`, an application `5xx` after headers, a deadline, or an ambiguous connection close after a mutation may have been accepted.
- `Idempotency-Key` is forwarded for order creation; it is not implemented in Gateway.

## Failure mapping

| Condition | Response | Operational meaning |
|---|---:|---|
| Invalid route/method | 404/405 | No upstream call |
| Invalid request shape/header | 400/413/415/422 | Client contract error |
| Missing/invalid token | 401 | Authentication failed |
| Insufficient scope/owner denied | 403 | Policy/owner denied |
| Rate limit exhausted | 429 | `Retry-After` is supplied |
| Required limiter/verifier unavailable | 503 | Protected policy cannot be safely enforced |
| Upstream deadline exceeded | 504 | Bounded dependency timeout |
| Upstream unavailable | 503 | Dependency unavailable |
| Unexpected Gateway failure | 500 | Generic safe Problem Details |

The Gateway preserves documented upstream status and safe Problem Details where doing so is part of the public contract; otherwise it maps to the table above. Internal stack traces, SQL, URLs, credentials, and hostnames are never returned.

## Rate-limit failure

Redis is not business state. If the configured policy cannot be evaluated, the route class determines behavior: authentication and mutations fail closed with `503`; health remains local; read behavior may fail closed where the policy says it is required. Every choice is observable and documented, never implicit.

## Backpressure

The implementation must bound request body size, concurrent upstream calls, idle connections, and response buffering. It must reject overload predictably and avoid unbounded queues. Backpressure metrics are low-cardinality and alertable.
