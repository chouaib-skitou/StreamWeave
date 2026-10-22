# STORY-029 — Validate and recover Orders

## Outcome

Prove the service remains correct under operational failure and can be restored without duplicate financial effects.

## Acceptance criteria

- [ ] PostgreSQL backup/PITR, Kafka replay, Outbox/DLQ repair, and Saga reconciliation runbooks are executable.
- [ ] Database, Kafka, Redis, pod eviction, duplicate event, and downstream timeout drills are automated or recorded.
- [ ] Security scan has no unresolved critical/high findings.
- [ ] Application test coverage is at least 85% with race detection.
- [ ] Release, rollback, and artifact verification evidence is attached.

## Test plan

Integration, E2E, chaos, load, security, recovery, and release workflow tests.

## API and event changes

Validate all documented REST responses, command/event schemas, replay behavior, and asynchronous status semantics. Recovery must preserve original IDs and Inbox/Outbox deduplication.

## Data and failure behavior

Exercise PITR, migration rollback boundaries, broker replay, DLQ repair, lease recovery, duplicate delivery, dependency timeout, and compensation failure without duplicate payment effects.

## Observability and security

Attach dashboard, alert, redaction, trace, and runbook evidence. Validate TLS, secret handling, least privilege, image provenance, and critical/high security findings before release.

## Dependencies and out of scope

Depends on all Orders implementation stories, platform CI, and service-owner contracts. Production approval remains gated on real environment evidence; documentation alone does not claim runtime readiness.
