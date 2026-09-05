# Orders Consistency and Failure Policy

## Local consistency

Order state, item snapshots, Saga transition, Inbox/Outbox writes, and idempotency records use PostgreSQL transactions. Optimistic aggregate versioning and unique constraints protect concurrent requests. Default isolation is `READ COMMITTED`; row locks or compare-and-swap are selected for each invariant.

## Cross-service consistency

Kafka provides at-least-once eventual consistency. The system does not claim distributed exactly-once delivery. Correctness comes from durable state guards, deterministic operation IDs, Inbox deduplication, Outbox publication, and compensation.

## Failure matrix

| Failure | Client-visible behavior | Durable response |
|---|---|---|
| PostgreSQL unavailable | `503` | no acceptance; no event |
| Kafka unavailable after local commit | `202` remains valid | Outbox backlog and alert |
| Relay crash after publish | none | duplicate safe through Inbox |
| Inventory rejects | order eventually `FAILED` | no payment command |
| Payment fails | order `FAILED` | release inventory command |
| Payment timeout | `PENDING` until fact/reconciliation | same operation ID |
| Fulfillment fails | retry, then refund/release | `FAILED` or review state |
| Redis unavailable | configured protected routes fail closed | correctness unchanged |
| Duplicate/out-of-order event | no duplicate side effect | Inbox/state guard |
| Poison event | no infinite retry | DLQ and operator runbook |

## Rollout compatibility

Database migrations are expand/migrate/contract. New consumers must tolerate old event fields, and old consumers must tolerate additive new fields. Rollback never rewinds committed business facts; it deploys a compatible binary and resumes durable work.
