# Orders Observability Design

## Metrics

Use low-cardinality labels only (`route`, `method`, `status`, `operation`, `outcome`, `dependency`, `reason_class`):

```text
orders_http_requests_total
orders_http_request_duration_seconds
orders_http_in_flight_requests
orders_idempotency_replays_total
orders_idempotency_conflicts_total
orders_status_transitions_total
orders_saga_in_progress
orders_saga_failed_total
orders_saga_step_duration_seconds
orders_outbox_pending_events
orders_outbox_oldest_event_age_seconds
orders_outbox_publish_failures_total
orders_inbox_duplicates_total
orders_consumer_failures_total
orders_consumer_lag
orders_dlq_messages_total
orders_db_pool_wait_seconds
orders_compensation_failures_total
```

Never use order IDs, customer IDs, email addresses, message IDs, raw URLs, keys, or arbitrary error text as labels.

## Traces

Propagate W3C trace context from Gateway through the local transaction, Outbox relay, Kafka headers, Inbox transaction, downstream command, response event, and compensation. Span attributes use route templates, operation class, message type, dependency, status, and sanitized failure class.

## Logs

Structured JSON includes timestamp, service, level, operation, route, request ID, trace/span IDs, correlation ID, order ID only where operationally justified, aggregate version, outcome, and bounded error code. It excludes body, address, email, bearer tokens, cookies, payment data, and full Kafka payloads.

## Alerts and dashboard

Shared Grafana panels cover API errors/latency, Saga age/failures, Outbox backlog/oldest age, Inbox duplicates, consumer lag, DLQ growth, compensation failures, database pool pressure, and dependency health. Alerts link to the Orders runbooks and use sustained thresholds to avoid paging on one transient event.

Initial alert thresholds are deliberately explicit and environment-tunable:

| Signal | Alert condition | For | Runbook |
|---|---|---:|---|
| API 5xx ratio | `> 2%` of requests | 10m | `orders-service-operations.md` |
| API p95 latency | `> 500ms` | 10m | `orders-service-operations.md` |
| Outbox oldest age | `> 60s` | 5m | `orders-outbox-backlog.md` |
| Saga failures | `> 5` failures | 10m | `orders-saga-recovery.md` |
| Consumer lag | `> 1,000` messages | 10m | `orders-consumer-dlq.md` |
| DLQ growth | any new message | 5m | `orders-consumer-dlq.md` |
| Compensation failures | any failure | 5m | `orders-saga-recovery.md` |
| DB pool wait p95 | `> 1s` | 5m | `orders-database-recovery.md` |

Paging thresholds are reviewed against traffic and SLOs before production activation; alert rules must be provisioned centrally with the shared monitoring stack, not duplicated inside the Orders deployment.
