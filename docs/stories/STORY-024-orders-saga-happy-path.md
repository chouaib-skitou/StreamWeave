# STORY-024 — Orchestrate the happy-path Saga

## Outcome

Progress a committed order through inventory reservation, payment authorization, confirmation, and fulfillment.

## Acceptance criteria

- [ ] Commands use explicit topics, operation IDs, and order partition keys.
- [ ] Inventory success precedes payment authorization.
- [ ] Payment success precedes `OrderConfirmed`.
- [ ] Fulfillment is requested only after confirmation prerequisites.
- [ ] Duplicate and late facts do not repeat business effects.

## Test plan

Contract, integration, duplicate-event, stale-version, and full happy-path E2E tests.

## API and event changes

The API returns `PENDING` after the initial commit. The Saga publishes reserve, authorize, and fulfillment commands in order, then publishes `OrderConfirmed` and `OrderCompleted` only after their prerequisites are durably observed.

## Data and failure behavior

Persist one Saga step and operation ID per downstream action. Reject duplicate, stale, late, malformed, and wrong-order facts without advancing the aggregate; verify payment amount and currency against the accepted quote.

## Observability and security

Trace each step with correlation and causation IDs, and measure step duration and outcome class. Use service-principal credentials and never put payment credentials or customer PII in commands.

## Dependencies and out of scope

Depends on Inventory, Payments, and Fulfillment contracts and the Outbox/Inbox implementation. Provider-specific pricing, payment, and shipping logic remain owned by those services.
