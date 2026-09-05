# Orders Security Incident Runbook

## Triggers

Suspected forged Gateway context, IDOR, unexpected command producer, token leakage, PII exposure, or abnormal cancellation/refund activity.

## Procedure

1. Restrict or stop the affected route/consumer without deleting evidence.
2. Rotate the Gateway service credential and revoke affected machine sessions through Identity.
3. Preserve sanitized logs, trace IDs, message IDs, audit records, and relevant hashes; never copy secrets or payload PII into the incident channel.
4. Review authorization, Inbox, Outbox, and downstream operation IDs for duplicate effects.
5. Repair or replay only after security approval and contract validation.

## Recovery

Run the security tests and artifact scan, verify NetworkPolicy/TLS/secret references, document impact, and reopen traffic gradually with enhanced alerts.
