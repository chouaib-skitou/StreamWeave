# ADR-020: Quote authority and immutable Orders price snapshots

- **Status:** accepted
- **Date:** 2026-09-05
- **Owners:** Platform architecture and Orders/Inventory

## Context

Clients must not supply prices, totals, discounts, or tax inputs that Payments later trusts. Orders needs a price and currency before it can create a durable financial workflow, while Inventory already owns products and stock in the platform scope.

## Decision

Inventory is the initial authority for product availability and price quotes. Orders requests a versioned quote/reservation boundary, validates its detached EdDSA JWS signature, key ID, version, and expiry, and snapshots product ID, quantity, unit price in minor units, currency, quote ID, and quote version in `orders_db`. The client request contains product IDs, quantities, and requested currency only.

Payments receives the immutable Orders total and currency through a command; it remains authoritative for authorization, capture, refund, and payment state. Orders stores no cardholder data.

## Consequences

- Existing price snapshots remain stable when catalog prices change.
- Inventory must implement the quote contract in `docs/design/orders-pricing-boundary.md` before the Orders integration story is accepted.
- A dedicated Pricing service is deferred until there is evidence that Inventory ownership is insufficient.
