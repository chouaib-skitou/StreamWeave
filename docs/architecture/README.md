# Architecture

This directory contains the platform and service architecture. Start with `docs/project_context.md`, then read the relevant system, service, data, API, event, Saga, security, observability, and scaling documents as they are added.

Decision records live in [`decisions/`](decisions/README.md). The project follows Clean/Hexagonal Architecture inside each independently deployable Go service and database-per-service ownership across the platform.

The Identity Service architecture is documented in [`identity-service.md`](identity-service.md) and is governed by the authentication and authorization ADRs under [`decisions/`](decisions/README.md).

The Gateway architecture is documented in [`gateway-service.md`](gateway-service.md). Gateway-specific boundary decisions are [ADR-015](decisions/ADR-015-gateway-public-edge.md), [ADR-016](decisions/ADR-016-gateway-auth-context.md), and [ADR-017](decisions/ADR-017-gateway-limits-and-failure-policy.md).
