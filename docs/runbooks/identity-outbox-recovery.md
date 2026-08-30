# Runbook: Identity Outbox Recovery

## Scope

Recover security-audit publication after Kafka outage, relay failure, or a crash after publication.

## Procedure

1. Confirm the Identity database is healthy and inspect pending rows, attempts, last error class, and oldest creation time.
2. Confirm the Kafka topic and producer credentials are available.
3. Resume the relay with bounded retry and backoff; keep `FOR UPDATE SKIP LOCKED` claiming enabled.
4. If a row may already have been published, replay it with the same event ID rather than creating a new audit fact.
5. Confirm consumers deduplicate by event ID and inspect the backlog until it returns below the alert threshold.
6. Record the incident window and any rows repaired without storing payload secrets.

## Safety rules

Never delete pending audit rows to hide backlog. Never edit a published event's identity or payload in place. If the payload is invalid, quarantine it through the documented DLQ process and preserve the original event ID for investigation.

## Validation

Verify database audit count, outbox publication status, Kafka delivery, consumer deduplication, and correlation IDs before closing the incident.
