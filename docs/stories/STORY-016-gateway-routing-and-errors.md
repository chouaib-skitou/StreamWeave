# STORY-016 — Route upstream calls and normalize public errors

## User / system outcome

Clients receive stable, correlated responses while internal services remain private and bounded.

## Context

The Gateway must preserve honest asynchronous semantics and prevent retries from duplicating mutations.

## Acceptance criteria

- [ ] Given a valid request, when it is routed, then the owning service receives only the documented safe context.
- [ ] Given `POST /v1/orders`, when the request is valid, then `Idempotency-Key` is required, validated, and forwarded unchanged.
- [ ] Given `POST /v1/orders`, when `customer_id` is supplied, then Gateway forwards it as untrusted input and Orders verifies ownership before accepting the command.
- [ ] Given an upstream timeout/unavailability, when the response is produced, then a safe 503/504 Problem Details response includes `request_id`.
- [ ] Given a mutation or auth request, when transport failure occurs, then Gateway does not automatically retry it.
- [ ] Given an idempotent read that fails before response headers, when policy allows, then at most one bounded retry occurs.
- [ ] Given `202 Accepted` from Orders, when Gateway returns it, then it does not claim synchronous completion.

## API changes

Public error and timeout behavior is defined in the OpenAPI contract and `gateway-resilience.md`.

## Event changes

None.

## Data changes

None; Orders owns idempotency and durable response state.

## Failure cases

Connect timeout, header timeout, body timeout, upstream 4xx/5xx, ambiguous mutation disconnect, invalid Problem Details.

## Observability

Upstream outcome and duration by logical service and route template; retry count remains bounded.

## Security considerations

No internal hostname, stack trace, SQL, or upstream credential in a response.

## Test plan

Transport failure matrix, timeout tests, no-retry mutation tests, one-retry read tests, error-envelope contract tests, and idempotency-header tests.

## Dependencies

Orders API/idempotency contract, Identity facade contract, and shared RFC 9457 error conventions.

## Out of scope

Orders business state machine, saga processing, and durable idempotency storage.

## Validation evidence

Documentation gate: resilience policy reviewed against the blueprint's 202/idempotency/error invariants.
