# STORY-025 — Compensate failure and cancellation

## Outcome

Handle inventory/payment/fulfillment failures and customer cancellation through durable, idempotent compensation.

## Acceptance criteria

- [x] Inventory rejection does not issue payment authorization.
- [x] Payment failure releases inventory.
- [x] Shipment failure emits refund and release commands after the failure path.
- [x] Cancellation races have deterministic precedence.
- [ ] Compensation failure becomes `FAILED_REQUIRES_REVIEW`, alertable and repairable.

## Test plan

Failure matrix, timeout-after-success, cancellation race, duplicate compensation, and repair tests.

## API and event changes

Cancellation returns `202` and publishes `OrderCancellationRequested`; terminal `OrderCancelled` or `OrderFailed` facts are emitted only after required compensations are confirmed. Notification delivery is not part of the transaction.

## Data and failure behavior

Persist compensation steps, deadlines, attempts, and operation IDs. Treat timeouts as unknown, reconcile before retrying, and quarantine contradictory outcomes. A failed compensation enters internal `FAILED_REQUIRES_REVIEW` and never silently reports success.

## Observability and security

Page on compensation failure and expose bounded failure classes, Saga age, and repair status. Repair is authenticated, rate-limited, audited, and cannot force arbitrary public status changes.

## Dependencies and out of scope

Depends on refund/release contracts and the recovery runbook. The current implementation persists
release/refund completion flags and uses the original Saga identity for cancellation commands.
Per-step repair tooling and the `FAILED_REQUIRES_REVIEW` workflow remain gated on the recovery
implementation and operator authorization design. Post-dispatch cancellation policy is deferred
to the Fulfillment contract and must be explicitly gated before implementation.
