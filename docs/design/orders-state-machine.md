# Orders State Machine

## Public states

| State | Meaning | Terminal | Allowed next states |
|---|---|---:|---|
| `PENDING` | Local order accepted; Saga is running | No | `CONFIRMED`, `CANCEL_REQUESTED`, `FAILED` |
| `CANCEL_REQUESTED` | Cancellation accepted locally; compensation is running | No | `CANCELLED`, `FAILED` |
| `CONFIRMED` | Inventory and payment prerequisites succeeded; fulfillment requested | No | `COMPLETED`, `CANCEL_REQUESTED`, `FAILED` |
| `COMPLETED` | Fulfillment reported completion | Yes | none |
| `CANCELLED` | Cancellation and required compensation completed | Yes | none |
| `FAILED` | Business workflow failed and is not progressing automatically | Yes for public state | none |

## Internal Saga phases

`INVENTORY_RESERVATION_PENDING`, `INVENTORY_RESERVED`, `PAYMENT_AUTHORIZATION_PENDING`, `PAYMENT_AUTHORIZED`, `FULFILLMENT_PENDING`, `CANCELLATION_PENDING`, `COMPENSATING`, `FAILED_REQUIRES_REVIEW`, and `SUCCEEDED` are durable internal phases. They are not public API statuses.

## Guards

- Inventory must be reserved before `AuthorizePayment` is issued.
- Payment must be authorized before `CONFIRMED` is emitted.
- Fulfillment completion must reference the same order and Saga.
- A stale or duplicate event is acknowledged through Inbox but cannot change state.
- Cancellation wins over an unstarted next step; if a remote step may have succeeded, compensation reconciles by operation ID before terminal cancellation.
- A terminal `FAILED` caused by a business rejection is not silently changed by a late success event. An operator repair command is required.

## Concurrency

Every write carries `aggregate_version`. PostgreSQL uses a conditional update or row lock so only one transition wins. A losing concurrent command reloads state and returns the documented current-state result. Process-local locks are not correctness mechanisms.

## Cancellation precedence

| Race | Result |
|---|---|
| Cancel before inventory command accepted | Do not issue reserve; cancel locally |
| Cancel after reservation | Release inventory, then cancel |
| Cancel after payment authorization | Refund payment, release inventory, then cancel |
| Cancel after shipment created | Return `409` unless Fulfillment reports a cancellable shipment |
| Cancel after completion | Return `409`; terminal order remains completed |
