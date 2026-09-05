# Orders Service Operations Runbook

## Trigger

Use this runbook for elevated API 5xx/p95 latency, readiness loss, dependency failures, or a rollout that does not converge.

## Procedure

1. Check the Orders API and relay pod status, readiness/startup responses, deployment revision, and recent sanitized error classes.
2. Check PostgreSQL connectivity/pool pressure, Kafka relay health, Inventory/Payments/Fulfillment dependency health, and telemetry exporter status.
3. If the issue is dependency-related, keep liveness available but do not bypass readiness, authentication, idempotency, or TLS policy.
4. For a bad rollout, pause the rollout and return to the last verified immutable image digest; never mutate the database with a destructive down migration.
5. Confirm Outbox age, Inbox failures, active Saga count, and duplicate/compensation metrics before reopening traffic.
6. Record request, trace, correlation, deployment, and sanitized failure references in the incident; do not copy payloads or credentials.

## Safety and exit criteria

Do not retry mutations at Gateway or force a public status. Recovery is complete only when readiness is healthy, API error/latency alerts clear for 15 minutes, no new DLQ or compensation failures appear, and the on-call operator records the evidence.
