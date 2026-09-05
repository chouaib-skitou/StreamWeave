# STORY-020 — Implement order domain and state machine

## Outcome

Represent an order with safe money, immutable price snapshots, aggregate versioning, and legal lifecycle transitions.

## Acceptance criteria

- [ ] Empty items, invalid IDs/currency, overflow, excessive quantities, and excessive totals are rejected.
- [ ] Public and internal statuses match the approved state machine.
- [ ] Terminal states cannot regress.
- [ ] Confirmation requires inventory and payment prerequisites.
- [ ] Concurrent transitions cannot overwrite a newer aggregate version.

## Test plan

Table-driven domain tests, property tests for money bounds, and concurrent transition tests.

## API and event changes

Define domain commands and results consumed by the API and Saga; produce no transport-specific types. Legal transitions map to the Orders fact schemas in `contracts/events/`.

## Data and failure behavior

Persist integer minor-unit money, immutable quote metadata, and aggregate versions. Reject invalid transitions, overflow, mismatched currency, stale versions, and terminal-state mutations without partial state changes.

## Observability and security

Record bounded transition and rejection classes only. Enforce customer ownership at the application boundary and keep domain objects free of tokens, PII, and infrastructure dependencies.

## Dependencies and out of scope

Depends on the PRD, state machine, pricing boundary, and event schemas. HTTP handlers, PostgreSQL repositories, and Kafka clients are out of scope here.
