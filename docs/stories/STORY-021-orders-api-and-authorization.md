# STORY-021 — Expose the Orders API securely

## Outcome

Implement the versioned Orders REST contract behind Gateway with ownership-aware authorization and honest asynchronous responses.

## Acceptance criteria

- [ ] Create and cancel return `202` only after local commit and require `Idempotency-Key`.
- [ ] Get/list enforce self ownership or explicit operator permissions.
- [ ] Cursors are opaque, bounded, actor/filter/sort bound, and stable.
- [ ] Errors use RFC 9457 Problem Details without enumeration or topology leakage.
- [ ] Gateway and Orders use `usr_...` customer references consistently.

## Test plan

OpenAPI contract tests, IDOR tests, pagination tests, body/header limits, and error mapping tests.

## API and event changes

Implement the create, get, list, and cancel endpoints from `contracts/openapi/orders.openapi.yaml`. Create and cancel append their intent facts only after the local transaction commits; reads emit no events.

## Data and failure behavior

Use durable idempotency for mutations and actor-bound opaque cursors for listing. Return generic RFC 9457 errors for authorization, validation, dependency, and conflict failures without revealing resource existence.

## Observability and security

Measure route/status/operation classes with request and trace correlation. Accept only Gateway-trusted actor context, enforce self versus operator policy, and never log bearer tokens, cursor contents, or request bodies.

## Dependencies and out of scope

Depends on the Gateway trust contract, Orders idempotency port, and OpenAPI contract. Saga execution and persistence implementation are delivered by adjacent stories.
