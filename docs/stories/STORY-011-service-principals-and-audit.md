# STORY-011 — Issue Service Tokens and Publish Audit Events

## User / system outcome

As a service process, I can obtain a scoped machine token without using a human bearer token; as an operator, I can trace the security action.

## Context

The blueprint requires separate service principals, machine audiences, short TTLs, and safe audit context.

## Acceptance criteria

- [ ] Given an active principal and valid credential, when the internal token endpoint is called, then a short-lived machine token with allowed scopes is returned.
- [ ] Given requested scopes exceeding the principal's grants, when issuance is requested, then `403` is returned and no token is issued.
- [ ] Given a valid machine-token response, then it contains no refresh token and expires after the documented five-minute default.
- [ ] Given a human token or public gateway route, when the machine endpoint is attempted, then it is rejected.
- [ ] Given a credential rotation, when the replacement is activated and the overlap window ends, then the previous credential is retired and both transitions are auditable.
- [ ] Given any Identity security action, when durable state changes, then its audit row and outbox event commit atomically.
- [ ] Given Kafka publication after an outbox crash, when a duplicate event is consumed, then the consumer can deduplicate by event ID.

## API changes

Implement internal-only `POST /v1/auth/service-token`.

## Event changes

Implement `commerce.security.audit.v1` using the versioned envelope and sanitized action payload.

## Data changes

Implement service principals, audit records, outbox rows, retry state, and indexes.

## Failure cases

Disabled principal, invalid credential, scope escalation, Kafka outage, relay crash, duplicate publication, and sanitized retry failure.

## Observability

Measure issuance, outbox backlog, publication attempts, duplicates, and relay failures with correlation IDs.

## Security considerations

Credentials are hashed or secret-referenced, never returned/logged; human bearer tokens never enter Kafka payloads. Follow the [service credential rotation runbook](../runbooks/identity-service-credential-rotation.md).

## Test plan

Internal-route, credential, audience, scope, outbox transaction, duplicate-event, and contract tests.

## Out of scope

Cloud workload identity and mandatory mTLS; both remain replaceable provider/deployment boundaries.

## Dependencies

STORY-006, STORY-008, ADR-011, ADR-013, ADR-014, and the security audit event schema.

## Validation evidence

Attach event-schema validation and duplicate-publication test evidence.
