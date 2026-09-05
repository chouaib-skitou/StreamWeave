# Orders Domain Model

## Aggregate

`Order` is the consistency boundary. It contains immutable identity and price snapshots, item lines, public status, aggregate version, and bounded failure metadata. A database row is not the domain model: transitions are performed through named use cases and guards.

```text
Order
├── order_id: ord_...
├── customer_id: usr_...
├── currency: ISO-4217
├── items[]
│   ├── product_id: prd_...
│   ├── quantity: 1..1000
│   ├── unit_price_minor: integer
│   ├── line_total_minor: integer
│   └── quote_id/version
├── total_minor: integer
├── public_status
├── aggregate_version
└── created/updated timestamps
```

## Value-object rules

- External IDs are UUIDv7 or ULID with stable prefixes.
- Money is `(amount_minor, currency)`; no floats, implicit currency conversion, or client total.
- Quantity multiplication checks overflow and configured maximum order total.
- Item lines are unique by product in the accepted request, unless a future pricing decision explicitly permits duplicates.
- Timestamps are RFC3339 UTC and assigned by the service clock.

## Commands

Application commands are `CreateOrder`, `RequestCancellation`, `ApplyInventoryResult`, `ApplyPaymentResult`, `ApplyFulfillmentResult`, and `RepairSaga`. Each command names its actor, correlation, causation, expected version, and operation ID where applicable.

## Domain events

The aggregate emits `OrderCreated`, `OrderCancellationRequested`, `OrderConfirmed`, `OrderCancelled`, `OrderFailed`, and `OrderCompleted`. Events describe a committed fact; they do not ask another service to perform work.

## Invariants

1. Empty orders are rejected.
2. A customer may not mutate another customer's order without an explicit operator permission.
3. `CONFIRMED` requires inventory reservation and payment authorization facts.
4. Terminal states cannot transition to an active state.
5. An event with an older aggregate version cannot overwrite newer state.
6. Every transition is auditable by event ID, actor class, correlation ID, and reason code without storing secrets.
