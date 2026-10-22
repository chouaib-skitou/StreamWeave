# STORY-023 — Implement Outbox and Inbox processing

## Outcome

Make event publication and consumption reliable under crashes, duplicates, retries, and replay.

## Acceptance criteria

- [ ] Aggregate changes and Outbox writes commit atomically.
- [ ] Inbox uniqueness is `(consumer_group, message_id)`.
- [ ] Relay leases recover after worker death and may publish duplicates safely.
- [ ] Consumers acknowledge only after state and resulting Outbox changes commit.
- [ ] Retry budgets and DLQ records are durable and observable.

## Test plan

Relay crash-window, duplicate delivery, out-of-order, poison message, and replay tests.

## API and event changes

Publish committed Orders facts and downstream commands through the Outbox; consume dependency facts through Inbox. Preserve envelope correlation, causation, aggregate version, Saga ID, and deterministic operation IDs.

## Data and failure behavior

Lease relay rows durably, retry transient errors with bounded backoff, and route permanent or exhausted failures to DLQ. A crash between publication and acknowledgement may duplicate a message but never duplicate a business effect.

## Observability and security

Expose backlog, age, attempts, duplicate, lag, and DLQ metrics. Redact payloads and credentials; quarantine invalid schemas, producers, versions, and transitions with a sanitized reason.

## Dependencies and out of scope

Depends on Kafka envelope schemas, PostgreSQL Inbox/Outbox tables, and the domain transition port. Downstream service business behavior and broker provisioning are out of scope.
