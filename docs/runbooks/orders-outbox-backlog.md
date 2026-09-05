# Orders Outbox Backlog Runbook

## Trigger

Alert: `orders_outbox_oldest_event_age_seconds` exceeds the environment threshold or pending rows grow continuously.

## Procedure

1. Confirm Orders API, relay, PostgreSQL, Kafka, and telemetry health.
2. Check whether leases are stale, Kafka is unavailable, or publishing errors share a failure class.
3. If relay replicas are stuck, restart one replica within the drain policy; never delete pending rows.
4. Restore Kafka connectivity or credentials, then verify backlog age decreases.
5. Investigate rows exceeding retry budget; quarantine to DLQ only through the documented operator command.

## Safety

Never mark an event published without a successful Kafka acknowledgement. Duplicate publication is safe only because consumers use Inbox and operation IDs.
