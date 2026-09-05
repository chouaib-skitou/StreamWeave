# STORY-022 — Persist Orders and durable idempotency

## Outcome

Persist order state, items, Saga state, and idempotency records in `orders_db` with atomic concurrency behavior.

## Acceptance criteria

- [ ] Migrations use expand/migrate/contract compatibility.
- [ ] `(actor, operation, key)` is unique and stores a canonical request hash.
- [ ] Same-key replay returns the original response; conflicting reuse returns `409`.
- [ ] Concurrent first requests create at most one order.
- [ ] Redis outage cannot change correctness.

## Test plan

PostgreSQL integration tests, transaction rollback tests, concurrent request tests, and migration validation.

## API and event changes

Persist the canonical request hash and replay the original bounded response for an identical actor/operation/key tuple. No duplicate `OrderCreated` fact may be produced.

## Data and failure behavior

Create the orders, items, Saga, Inbox, Outbox, and idempotency tables with the documented indexes and constraints. Roll back all local writes on transaction failure; Redis availability must not affect correctness.

## Observability and security

Measure replay, conflict, lock-wait, migration, and rollback classes without storing secrets or unbounded response data. Protect customer references and enforce bounded retention.

## Dependencies and out of scope

Depends on the data model, migration policy, PostgreSQL transaction port, and API operation identities. Kafka relay behavior is implemented in STORY-023.
