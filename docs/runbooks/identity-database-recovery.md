# Runbook: Identity Database Recovery

## Scope

Recover Identity PostgreSQL while preserving users, sessions, roles, token-family state, audit records, and outbox correctness.

## Procedure

1. Confirm the outage and stop writes if the database is inconsistent or in recovery.
2. Check application readiness, connection-pool errors, migration state, and outbox backlog.
3. Restore the latest approved backup to the Identity database only; never merge it with another service database.
4. Apply only compatible migrations and verify constraints, indexes, role seeds, and session-family links.
5. Start Identity in a restricted mode, verify health, JWKS, login, refresh, and session revocation behavior.
6. Resume traffic and monitor authentication failures, refresh reuse, database latency, and outbox publication.

## Recovery notes

Refresh tokens and audit/outbox rows are durable Identity state. Redis data must not be restored as the source of truth. If audit rows were restored before their outbox rows, reconcile using the [outbox recovery procedure](identity-outbox-recovery.md) before claiming publication completeness.
