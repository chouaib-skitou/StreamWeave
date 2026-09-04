# ADR-015: Gateway is the single explicit public edge

- **Status:** accepted
- **Date:** 2026-09-04
- **Owners:** Platform architecture
- **Scope:** Gateway public HTTP boundary

## Context

StreamWeave contains independently deployable services. Clients need a stable public origin, while service ownership and internal topology must remain private. A generic reverse proxy would make route ownership, security, and failure behavior implicit.

## Decision

Gateway is the only public HTTP edge for the initial `/v1` API. It exposes an explicit route allowlist for Identity authentication/session operations and Orders order operations. It owns request IDs, tracing, coarse authentication policy, rate limits, routing, standardized errors, and access logs. It does not expose arbitrary upstream paths or internal service-token issuance.

## Consequences

- Clients depend on one versioned facade.
- Route policy is reviewable as a contract and testable without business services.
- New services require an explicit route and ownership decision.
- Gateway is not a general API gateway product; aggregation, uploads, GraphQL, and WebSockets require separate decisions.

## Alternatives rejected

- **Direct client-to-service access:** leaks topology and duplicates security policy.
- **Generic open proxy:** creates an authorization and SSRF boundary that cannot be safely reviewed.
- **Business logic in Gateway:** duplicates source-of-truth decisions and makes asynchronous workflows dishonest.

## Failure modes considered

- A route omitted from the allowlist is rejected locally rather than proxied.
- An upstream outage produces a bounded public error and does not expose topology.
- A route policy change is caught by route-matrix and contract tests before release.

## Validation

The decision is reflected by `gateway-route-matrix.md`, `gateway.openapi.yaml`, and STORY-014.
