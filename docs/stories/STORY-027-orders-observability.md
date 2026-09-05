# STORY-027 — Operate Orders visibly

## Outcome

Expose low-cardinality metrics, trace continuity, structured redacted logs, dashboards, alerts, and SLO evidence.

## Acceptance criteria

- [ ] API, Saga, Outbox, Inbox, consumer, DLQ, compensation, and DB-pool telemetry exists.
- [ ] Trace context survives HTTP → Kafka → downstream → compensation.
- [ ] PII, tokens, payloads, and IDs are not unbounded labels or logs.
- [ ] Every alert links to an Orders runbook.

## Test plan

Telemetry assertions, redaction tests, alert-rule validation, and dashboard smoke checks.

## API and event changes

Expose only the documented health and metrics surfaces; no business event is introduced by telemetry. Kafka trace headers remain compatible with the common envelope.

## Data and failure behavior

Do not add observability data to the business source of truth. Exporter failure degrades telemetry without changing order correctness, while readiness reflects dependencies according to the deployment contract.

## Observability and security

Use the shared Prometheus, Grafana, Loki, and Tempo stack with explicit Orders panels and alerts. Labels and logs exclude IDs, emails, tokens, cookies, payloads, and raw error text.

## Dependencies and out of scope

Depends on shared monitoring provisioning and Orders metric vocabulary. Vendor-specific dashboard hosting and long-term retention policy are platform concerns.
