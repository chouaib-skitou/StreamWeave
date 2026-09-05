# Orders Idempotency

## HTTP

The authoritative key is `(actor_id, operation, idempotency_key)`. The request fingerprint is a canonical JSON hash after validation and normalization. Same key and same fingerprint replays the stored response; same key with a different fingerprint returns `409`. An in-progress concurrent request waits up to 2 seconds for the owning transaction; if it is still in progress, it returns `409` with problem type `idempotency_in_progress` and `Retry-After: 1`. It never creates a second order.

Create and cancel use separate operation names. The key is required, bounded to 16–256 characters, never logged in full, and records expire 48 hours after the first committed response. The purge job retains aggregate, Saga, Inbox, Outbox, and audit history for the configured 90-day operational/reconciliation window; active Sagas are never purged.

## Messaging

Every downstream command carries a deterministic operation ID. Consumers persist `(consumer_group, message_id)` in Inbox. A relay crash after publish can create a duplicate message; receiving services must make reservation, authorization, refund, release, and shipment creation idempotent by operation ID.

## Ambiguous timeouts

If the client times out after the local commit, retry with the same key. If a downstream call times out after remote acceptance, Orders waits for the matching fact or retries the same operation ID. It never treats an unknown outcome as permission to create a new operation.

## Abuse controls

Keys are rate-limited and stored with bounded size and retention. Excessive unique keys, request bodies, items, and list filters are rejected. Redis may accelerate hot-key checks but a Redis outage cannot cause duplicate creation or bypass PostgreSQL uniqueness.
