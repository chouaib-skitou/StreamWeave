# ADR-014 — Security Audit Delivery

- Status: Accepted
- Date: 2026-08-29
- Owners: Project maintainer

## Context

Authentication and authorization actions must remain traceable when Kafka is unavailable or an outbox relay crashes. Audit data must not become a second channel for leaking credentials.

## Decision

Identity writes a sanitized `security_audit` row and a `commerce.security.audit.v1` outbox event in the same PostgreSQL transaction as the identity state change. The outbox relay publishes to the dedicated security audit topic with at-least-once semantics. Consumers deduplicate by event ID. The payload contains bounded action/outcome/actor/target fields and no passwords, tokens, private keys, raw email addresses, or bearer credentials.

## Alternatives considered

- Publish directly from the HTTP handler: rejected because Kafka failure would lose or reorder the audit fact relative to durable state.
- Log-only audit: rejected because logs are not the durable source of truth and are harder to replay.
- One topic per action: rejected because a versioned security audit stream keeps the contract surface bounded.

## Consequences

Audit publication can lag behind a successful request, but the outbox backlog makes the condition visible and recoverable. Consumers must tolerate duplicate events and must not assume synchronous publication.

## Failure modes and recovery

PostgreSQL failure prevents the state change. Kafka failure leaves the outbox row pending. Relay crash after publication permits a duplicate. Sanitization is enforced before persistence and again at serialization boundaries.

## Validation

Test atomic state-plus-outbox writes, Kafka outage, relay retry, duplicate publication, schema validation, forbidden sensitive fields, and correlation propagation.

## Revisit when

The platform adopts a regulated audit archive, centralized event governance, or a managed security information and event management pipeline.
