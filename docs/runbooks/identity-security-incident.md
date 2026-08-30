# Runbook: Identity Security Incident

## Scope

Respond to suspected credential compromise, refresh-token replay, signing-key exposure, or privilege escalation.

## Immediate containment

1. Preserve sanitized logs, audit records, traces, and correlation IDs.
2. Disable affected users or service principals.
3. Revoke affected session families and place known access-token JTIs on the emergency denylist.
4. If signing material may be exposed, rotate keys using the key-rotation runbook and invalidate the affected issuer/key policy.
5. Do not copy raw passwords, tokens, private keys, or exploit material into issues, commits, or chat.

## Investigation

Review login failures, refresh reuse, role mutations, service-token issuance, JWKS changes, and outbox publication. Correlate events without expanding sensitive data retention.

## Recovery

Reset affected credentials, restore explicit least-privilege roles, verify gateway and business-service authorization, and run the security regression suite before reopening access.

## Follow-up

Record a sanitized incident decision in an ADR or private security channel. Add a regression test and update this runbook if a new failure mode was discovered.
