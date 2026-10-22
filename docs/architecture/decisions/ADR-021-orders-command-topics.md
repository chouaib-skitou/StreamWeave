# ADR-021: Separate Orders command and event topics

- **Status:** accepted
- **Date:** 2026-09-05
- **Owners:** Platform architecture and messaging

## Context

The blueprint lists domain event topics but leaves commands and responses partially implicit. Treating a request for work as a fact event makes ownership, replay, ACLs, and observability unclear.

## Decision

Orders publishes commands to explicit domain command topics:

- `commerce.inventory.commands.v1`
- `commerce.payment.commands.v1`
- `commerce.fulfillment.commands.v1`

Orders publishes facts to `commerce.order.events.v1` and consumes facts from the corresponding domain event topics. Every message uses the versioned envelope, `order_id` partition key, `correlation_id`, `causation_id`, deterministic operation ID, and authenticated producer metadata.

## Consequences

- ACLs can distinguish command producers from event producers.
- Consumers can reject the wrong message class before domain processing.
- More schemas must be maintained, but compatibility and replay behavior become explicit.
