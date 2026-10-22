# Orders Pricing Boundary

## Decision

Inventory is the initial authority for product availability and price quotes. The client sends product IDs, quantities, and a requested ISO currency only. It never sends unit prices, discounts, tax, totals, or payment credentials.

Orders commits the initial order intent as `PENDING` and publishes `ReserveInventory`. Inventory returns a versioned quote/reservation result containing product references, accepted quantities, integer unit prices, currency, quote ID/version, expiry, and a detached JWS signature over the canonical quote payload. Orders verifies the signature against the Inventory public-key set, rejects expired or replayed quotes, snapshots the result in a local transaction, calculates the total, and only then publishes `AuthorizePayment`.

## Why this boundary

Orders needs a stable amount for Payment authorization, but duplicating catalog authority would create price drift. A dedicated Pricing service is unnecessary for the initial portfolio scope. Inventory already owns products and stock and can make the quote/reservation decision atomically in its own boundary.

## Rules

- Currency mismatch or expired/invalid quote fails the pending Saga without authorizing payment; the local order intent remains traceable.
- Prices are immutable snapshots on an accepted order.
- Integer multiplication checks overflow and maximum total.
- Payments receives only the Orders snapshot total and currency; Orders never stores PAN, CVV, tokens, or payment method secrets.
- Future promotions/taxes must be represented by a versioned pricing contract, never by client-calculated totals.

## Quote contract

- `quote_id`, `quote_version`, `quote_key_id`, `quote_expires_at`, and `quote_signature` are mandatory on `InventoryReserved`.
- `quote_signature` is a detached JWS using the Inventory-owned EdDSA key identified by `quote_key_id`. The signed canonical payload contains `order_id`, `reservation_id`, `quote_id`, `quote_version`, `currency`, accepted items, and `quote_expires_at`.
- Orders resolves Inventory's internal JWKS through an authenticated TLS endpoint and caches only public keys for the documented rotation overlap. An unknown key causes one bounded refresh, then fail-closed quarantine.
- The reservation operation ID is the command envelope `operation_id`; Inventory must return the same operation ID in the event envelope and produce one deterministic result for retries.
- `quote_expires_at` is checked against trusted UTC time before the quote is persisted. A late result is rejected and compensated through the Inventory release contract.

## Required future Inventory contract

The Inventory service documentation must preserve this quote/reservation contract, rejection codes, and behavior for a timeout after reservation success. Orders keeps the pricing port explicit and does not invent a generic catalog client. The quote boundary is asynchronous and does not block the initial local `PENDING` acceptance transaction.
