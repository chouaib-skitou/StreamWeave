# Runbook: Identity Redis Outage

## Expected behavior

Redis is disposable. PostgreSQL continues to own users, refresh sessions, role state, reset tokens, verification tokens, audit, and outbox data.

- Refresh-session correctness remains PostgreSQL-backed.
- Rate limiting follows the configured fail-closed policy for credential endpoints.
- Emergency JTI denylist checks fail closed at the gateway while revocation protection is enabled.
- Readiness and metrics expose the degraded dependency state.

## Procedure

1. Confirm Redis connectivity errors and affected routes.
2. Verify no application path is treating Redis as durable identity state.
3. Restrict or temporarily disable high-risk credential endpoints if the configured policy requires it.
4. Recover or replace Redis without restoring stale business/session truth from it.
5. Verify login rate limiting, refresh, logout, JTI revocation, and readiness after recovery.

## Evidence

Record the outage window, denied requests, recovered metrics, and confirmation that PostgreSQL session state was preserved.
