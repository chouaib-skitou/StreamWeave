# ADR-019: Canonical customer reference in Orders

- **Status:** accepted
- **Date:** 2026-09-05
- **Owners:** Platform architecture and Identity/Orders

## Context

The initial Gateway facade used `cus_...` examples while Identity issues `usr_...` subjects. No separate customer mapping authority exists. Keeping both formats without a contract would make ownership checks ambiguous and invite IDOR defects.

## Decision

Orders v1 uses the trusted Identity subject (`usr_...`) as `customer_id`. For self operations, Orders derives the effective customer from the authenticated Gateway actor context and rejects a mismatching request field. Operator operations may target another `usr_...` only with an explicit `orders:write:any`, `orders:read:any`, or `orders:cancel:any` permission.

Gateway and Orders contracts must use the same format. A future customer profile or tenant service may introduce a mapping through a versioned contract; it must not be inferred from email.

## Consequences

- No hidden cross-service lookup is required for ownership.
- Existing pre-implementation `cus_...` examples must be aligned before Orders implementation.
- The Identity subject becomes a visible external reference; its stability is therefore part of the v1 contract.
