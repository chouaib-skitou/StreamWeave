# ADR-022: Reconcile ambiguous downstream operations by operation ID

- **Status:** accepted
- **Date:** 2026-09-05
- **Owners:** Platform architecture and Orders

## Context

An HTTP or Kafka timeout can occur after Inventory, Payments, or Fulfillment accepted a command. Retrying with a new identifier can create duplicate reservations, authorizations, refunds, or shipments.

## Decision

Orders generates one deterministic operation ID per Saga step, retries that same operation, and uses the private authority lookup contract in [`orders-reconciliation.md`](../../design/orders-reconciliation.md) when the result remains unknown. The lookup is read-only from Orders' perspective; the normal fact event, Inbox, and aggregate transition remain the only mutation path.

## Consequences

- Downstream services must retain operation status for the replay/reconciliation window.
- Payment ambiguity is escalated to Payments/Finance before a terminal financial state is reported.
- Recovery remains idempotent and auditable, at the cost of explicit private lookup contracts.
