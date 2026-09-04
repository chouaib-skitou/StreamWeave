# STORY-014 — Establish Gateway foundation and explicit public contract

## User / system outcome

Clients have one versioned Gateway facade with an explicit route registry and consistent request identity.

## Context

Gateway is the public edge for the initial Identity and Orders routes. It must not become an open proxy or own business state.

## Acceptance criteria

- [x] Given a documented method/path, when a request arrives, then it is dispatched only to its declared owner.
- [x] Given an unknown path or method, when a request arrives, then no upstream call occurs and a safe 404/405 response is returned.
- [x] Given no valid request ID, when a request arrives, then Gateway generates one and returns it.
- [x] Given an invalid body size/content type, when a request arrives, then it is rejected before proxying.
- [x] Given the OpenAPI contract, when route tests run, then every registered public route is covered.

## API changes

Implement the routes in `contracts/openapi/gateway.openapi.yaml` and the policy in `docs/design/gateway-route-matrix.md`.

## Event changes

None. Gateway does not publish business events.

## Data changes

None. No Gateway database.

## Failure cases

Unknown route, unsupported method, invalid request ID, oversized body, unsupported content type, and malformed correlation headers.

## Observability

Route-template request counter, duration, status, request ID, trace ID, and bounded error code.

## Security considerations

No open proxy, no raw URL/body logging, no client-controlled internal headers, no secrets in configuration.

## Test plan

Route table unit tests, OpenAPI contract validation, malformed-header tests, body-limit tests, and black-box unknown-route tests.

## Dependencies

Gateway OpenAPI contract, route matrix, shared HTTP conventions, and CI contract validation.

## Out of scope

JWT verification, rate-limit implementation, business authorization, and Gateway code generation details.

## Validation evidence

Documentation gate: route matrix and OpenAPI contract reviewed against the blueprint and Identity/Orders ownership boundaries.
