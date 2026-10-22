# Orders Consumer Lag and DLQ Runbook

## Trigger

Alert: `orders_consumer_lag`, `orders_dlq_messages_total`, or consumer failure rate exceeds its sustained threshold.

## Procedure

1. Identify the consumer group, topic, partition, schema version, and sanitized failure class.
2. If dependency-related, repair the dependency and allow bounded retry to resume.
3. If schema/auth/domain-invalid, leave the message quarantined and open an incident; do not bypass validation.
4. Before replay, inspect the order, Saga step, Inbox record, and downstream operation ID.
5. Replay the original event ID through the controlled tool and verify no duplicate business effect.

## Safety

Replay is authorized, audited, rate-limited, and never edits public status directly.
