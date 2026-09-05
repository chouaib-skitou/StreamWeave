# Orders Reconciliation Contract

## Purpose

An acknowledged command can outlive the network request that delivered it. Orders must resolve that ambiguity without issuing a second payment, reservation, refund, or shipment operation.

## Required dependency interface

Every Inventory, Payments, and Fulfillment implementation exposes an internal, Gateway-inaccessible lookup keyed by the original `operation_id`:

| Authority | Internal lookup | Result states |
|---|---|---|
| Inventory | `GET /internal/v1/reservations/by-operation/{operation_id}` | `NOT_FOUND`, `PENDING`, `RESERVED`, `REJECTED`, `RELEASED` |
| Payments | `GET /internal/v1/authorizations/by-operation/{operation_id}` | `NOT_FOUND`, `PENDING`, `AUTHORIZED`, `FAILED`, `REFUNDED`, `REFUND_FAILED` |
| Fulfillment | `GET /internal/v1/shipments/by-operation/{operation_id}` | `NOT_FOUND`, `PENDING`, `CREATED`, `COMPLETED`, `FAILED` |

These endpoints are private service-to-service contracts, authenticated with short-lived machine tokens, mTLS or equivalent transport identity, strict timeouts, and audit context. They are never exposed through Gateway or returned to customers.

## Resolution policy

1. Retry the same command with the same `operation_id` within the bounded retry budget.
2. If the result remains unknown, call the owning lookup once per reconciliation interval and persist the observed state.
3. Apply the corresponding fact through the normal Inbox and aggregate-version path; a lookup never mutates Orders directly.
4. If the authority returns `NOT_FOUND` after the command acceptance deadline, quarantine the Saga for operator review rather than issuing a new operation blindly.
5. Payment authorization/refund ambiguity always escalates to Payments/Finance review before a terminal public status is changed.

## Contract invariants

- The same operation ID is unique at each dependency authority and is retained for the replay/reconciliation period.
- A lookup response includes `operation_id`, authority reference, status, currency/amount where financial, `updated_at`, and correlation ID.
- Orders validates the response against the expected command type and order ID; mismatches are quarantined.
- Recovery is idempotent, audited, rate-limited, and observable through the Saga runbook.
