# ADR-010 — RBAC and Explicit Scopes

- Status: Accepted
- Date: 2026-08-29
- Owners: Project maintainer

## Context

Gateway route checks alone cannot enforce ownership or business policy. A role name alone is also too coarse to express the permissions needed by different commerce actors.

## Decision

Identity stores roles and explicit scopes. Roles are convenience bundles; authorization decisions use scopes plus resource ownership. The gateway performs authentication and coarse scope checks. The owning application use case performs the final resource and business authorization check. `admin` is not an implicit bypass.

## Alternatives considered

- Role-only checks: rejected because roles do not express resource scope clearly.
- Central authorization service for every decision: deferred because it creates a synchronous dependency and hides domain ownership.
- Repository-level authorization: rejected as the primary layer because repositories should receive already-authorized criteria.

## Consequences

Scope changes affect newly issued tokens. Existing access tokens remain valid until expiry or emergency revocation. Every new scope requires an owning service, an endpoint/event impact review, and tests.

## Failure modes and recovery

Missing scopes and resource ownership failures return `403`. Unknown role/scope assignments are rejected. Role mutation is transactional, audited, and does not silently grant unrelated permissions.

## Validation

Test customer self-access, customer foreign-resource denial, support read-any, support refund denial, finance refund access, and admin role mutation.

## Revisit when

Policy becomes attribute-based, tenant-aware, or requires centralized policy evaluation across many services.
