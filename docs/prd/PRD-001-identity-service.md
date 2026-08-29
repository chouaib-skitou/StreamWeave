---
title: Identity Service
status: proposed
created: 2026-08-29
updated: 2026-08-29
---

# PRD: Identity Service

## 0. Document Purpose

This document defines the product and system requirements for the Identity Service before implementation. It is the implementation contract for the identity boundary and feeds the Identity architecture, API contract, ADRs, stories, tests, and operational runbooks. The governing source is the local blueprint `01_event_driven_ecommerce_order_management_platform.md`; this PRD makes its authentication and authorization decisions testable without changing the platform's service boundaries.

## 1. Vision

The Identity Service provides a durable, independently deployable authentication boundary for the commerce platform. It answers who a user or service principal is, issues short-lived credentials, manages sessions and roles, and publishes security audit facts without exposing secrets or coupling business services to identity storage.

The service must remain correct when clients retry, refresh tokens concurrently, lose network responses, or encounter Redis failure. PostgreSQL is authoritative for users, credentials, sessions, roles, token families, reset tokens, verification tokens, and audit records. Redis is disposable and is used only for short-lived emergency access-token revocation and rate limiting.

## 2. Target Users and Actors

### 2.1 Jobs to Be Done

- **Customer:** sign in once and use a short-lived access token to create and inspect owned orders.
- **Support operator:** authenticate with explicit permissions and inspect or cancel orders allowed by policy.
- **Finance operator:** authenticate with refund permissions without receiving unrelated administrative powers.
- **Platform administrator:** manage roles through explicit permissions and auditable actions.
- **Service process:** obtain a short-lived machine token without reusing a human bearer token.
- **Gateway and business services:** verify identity from JWKS and enforce their own authorization decisions.

### 2.2 Non-Users in v1

- Public production self-signup without anti-abuse and email-delivery operations.
- External identity providers, social login, SAML, and OIDC federation.
- Real email delivery; the verification and reset mailer is simulated.

### 2.3 Key User Journeys

- **UJ-1. A customer signs in and places an order.** The customer submits credentials, receives an access token and opaque refresh token, presents the access token through the gateway, and is denied if the token is expired, forged, revoked, or issued for another audience.
- **UJ-2. A customer refreshes a session safely.** The client exchanges a refresh token, receives a new access token and refresh token, and the previous refresh token becomes unusable. Reuse of a rotated token revokes the complete session family.
- **UJ-3. An administrator changes a role.** The administrator assigns or removes a role, the action is authorized by explicit administrative permission, and a newly issued access token reflects the updated scope set.
- **UJ-4. A service authenticates machine-to-machine.** An approved service principal requests a short-lived token for a separate machine audience and receives only the requested allowed scopes.
- **UJ-5. An operator recovers a security incident.** The operator revokes a session or token family, the service records an audit event, and emergency access-token revocation is enforced for the remaining token lifetime.

## 3. Glossary

- **Access token** — a short-lived EdDSA JWT used by gateways and services to authenticate a request.
- **Refresh token** — a 256-bit opaque secret returned to a client and stored only as a SHA-256 hash.
- **Session family** — all rotated refresh-token sessions descended from one login session.
- **Scope** — a narrow permission such as `orders:read:self` or `payments:refund`.
- **Role** — a named set of scopes assigned to a user.
- **Service principal** — a non-human identity allowed to obtain a machine token.
- **JWKS** — the public JSON Web Key Set used to verify access-token signatures.
- **Security audit event** — a versioned, non-secret record of an identity or authorization action.
- **Demo registration** — the development-only registration endpoint described by the blueprint.

## 4. Features and Functional Requirements

### 4.1 User and Credential Lifecycle

The service manages users and password credentials in its own PostgreSQL database. Demo registration is explicitly separated from a production signup product.

#### FR-1: Demo registration

The system shall expose `POST /.well-known/register` only when demo registration is enabled by configuration.

**Consequences:**

- A valid request creates a user with a normalized unique email and an Argon2id password hash.
- The endpoint never returns a password, password hash, refresh token, or private key.
- The endpoint is disabled by default in production configuration.
- Duplicate normalized email handling does not reveal whether a production account exists; demo mode may return a documented conflict response.

#### FR-2: Credential verification

The system shall verify passwords using Argon2id and return one generic invalid-credentials outcome for unknown users and incorrect passwords.

**Consequences:**

- Passwords are never logged, persisted in plaintext, or included in audit metadata.
- Login failure responses do not disclose whether an email exists.
- Successful authentication creates a durable session family and emits a security audit event.

#### FR-3: Password reset and email verification

The system shall support hashed, single-use, expiring password-reset and email-verification tokens with simulated delivery.

**Consequences:**

- Reset requests return the same accepted response whether or not the email exists.
- A consumed, expired, or unknown token cannot be used.
- Password reset revokes the user's active session families.
- Email verification updates only the owning user's verification state.

### 4.2 Access Tokens and Sessions

#### FR-4: Human access-token issuance

The system shall issue short-lived EdDSA JWT access tokens containing issuer, audience, subject, expiry, issued-at, token ID, roles, and scopes.

**Consequences:**

- Access-token lifetime is 10 minutes by default.
- The signing algorithm is an explicit Ed25519/EdDSA allowlist.
- Tokens use a stable key ID and are verifiable through the JWKS endpoint.
- Human tokens use the platform API audience and cannot be used as service credentials.

#### FR-5: Refresh-token rotation

The system shall issue opaque refresh tokens and rotate them on every successful refresh.

**Consequences:**

- Only a SHA-256 hash of a refresh token is stored in PostgreSQL.
- The previous token is atomically marked rotated and linked to its replacement.
- Concurrent use of the same refresh token produces at most one successful rotation.
- Reuse of a rotated token revokes the complete session family and returns a generic unauthorized response.

#### FR-6: Session revocation

The system shall support logout of the current session, logout of all user sessions, session listing, and administrator/operator session revocation according to explicit permissions.

**Consequences:**

- Logout is idempotent for an already revoked session.
- Logout-all revokes every active family for the authenticated user.
- Revoked access-token JTIs may be placed in the Redis denylist with a TTL no longer than the token's remaining lifetime.
- PostgreSQL remains sufficient to enforce refresh-session revocation when Redis is unavailable.

### 4.3 JWKS and Token Verification Contract

#### FR-7: Public key discovery

The system shall expose `GET /.well-known/jwks.json` with active and still-valid public signing keys.

**Consequences:**

- Private signing material is never returned.
- Key IDs remain stable for the lifetime of tokens signed with them.
- Old public keys remain published until all tokens signed with them can expire or are otherwise retired by policy.
- Invalid key configuration prevents readiness rather than silently issuing unverifiable tokens.

#### FR-8: Claims and issuer policy

The system and consuming gateway shall validate signature, issuer, audience, expiry, not-before when present, algorithm allowlist, and emergency JTI revocation.

**Consequences:**

- A valid signature with the wrong issuer or audience is unauthorized.
- An algorithm selected by an untrusted token header is never accepted.
- A missing or malformed required claim is unauthorized.

### 4.4 RBAC and User Administration

#### FR-9: Explicit scopes and roles

The system shall assign roles that resolve to explicit scopes and shall never treat `admin` as an implicit bypass.

**Consequences:**

- Customer, support, warehouse, finance, and admin role scopes match the blueprint baseline.
- Role changes are audited and affect newly issued tokens.
- Business services perform resource ownership checks after gateway scope checks.

#### FR-10: User and role endpoints

The system shall expose user retrieval and role assignment/removal endpoints protected by explicit administrative permissions.

**Consequences:**

- A caller cannot assign or remove roles without authorization.
- Role names and scopes are validated against the service's durable catalog.
- All successful role changes and denied administrative actions are observable without sensitive metadata.

### 4.5 Service-to-Service Identity

#### FR-11: Machine token issuance

The system shall issue short-lived machine tokens to approved service principals through an internal-only client-credentials endpoint.

**Consequences:**

- Service principals have separate audiences and machine scopes.
- Human access tokens are rejected by machine-only endpoints.
- Service credentials are stored only as hashes or secret references; the raw credential is never logged or returned.
- The requested scope list is required; every requested scope must be assigned to the principal or the service returns `403` without issuing a token.
- Machine access tokens have a five-minute default lifetime and never receive a refresh token.
- The gateway does not expose the machine-token endpoint as a public customer route.

### 4.6 Security Audit

#### FR-12: Durable and published audit facts

The system shall record and publish login, refresh, logout, session revocation, user status changes, role changes, password-reset, email-verification, service-credential rotation, and service-token actions as security audit events.

**Consequences:**

- Audit records contain actor and target identifiers only when known and never contain passwords, bearer tokens, refresh tokens, reset tokens, or private key material.
- User status changes and service-credential rotations are included in the bounded audit action catalog.
- Audit rows and their outbox records are committed atomically.
- Duplicate publication is safe for consumers through the event ID.
- Audit publication failure does not make a committed authentication result appear successful without an operationally visible outbox backlog.

## 5. Cross-Cutting Non-Functional Requirements

- **NFR-1 Security:** Argon2id is used for passwords; refresh, reset, and verification tokens are opaque and hashed; signing keys are asymmetric and private to Identity.
- **NFR-2 Privacy:** Responses and logs do not disclose account existence, credentials, token values, or sensitive network data. IP data is reduced to a documented prefix hash when retained for security analysis.
- **NFR-3 Availability:** Refresh and session correctness do not depend on Redis availability. Redis outage may degrade rate limiting or emergency revocation according to the fail-closed policy in the architecture document, but must not lose durable identity state.
- **NFR-4 Performance:** Under the initial local production-like profile, authentication endpoints target p95 latency below 500 ms at 25 requests/second, excluding dependency outages. Argon2id cost must be benchmarked rather than disabled to meet an arbitrary latency target.
- **NFR-5 Reliability:** Every durable state change that emits an audit event uses the Identity PostgreSQL transaction and outbox. Consumers must tolerate duplicate delivery.
- **NFR-6 Observability:** HTTP requests, authentication outcomes, session transitions, database operations, outbox publication, and revocation checks expose structured logs, low-cardinality metrics, and OpenTelemetry traces.
- **NFR-7 Operations:** The service exposes live and ready health endpoints, supports graceful shutdown, uses bounded dependency timeouts, and provides documented key rotation, database recovery, Redis outage, and security-incident procedures.
- **NFR-8 Deployment:** The service runs as a non-root container with a read-only filesystem where compatible, explicit resource limits, secret references, network policy, liveness/readiness/startup probes, and a versioned migration process.
- **NFR-9 Compatibility:** The OpenAPI contract and security audit event schema are versioned. Breaking changes require a new version and a migration plan.

Credential issuance is fail-closed when its required Redis rate limiter is unavailable: login, demo registration, password-reset request, and service-token issuance return dependency-unavailable without changing credentials. Refresh, logout, and durable session inspection remain PostgreSQL-backed; emergency JTI verification remains fail-closed.

## 6. Non-Goals

- Real email or SMS delivery.
- External identity providers and federation.
- Customer-facing production signup without anti-abuse controls.
- Passwordless or WebAuthn authentication in the first implementation.
- Authorization decisions for orders, payments, inventory, or fulfillment; those remain in their owning application use cases.
- Storing user data in Redis or sharing the Identity database with another service.

## 7. MVP Scope

### 7.1 In Scope

- Demo registration behind configuration.
- Login, access-token issuance, JWKS, refresh rotation, logout, logout-all, and session listing.
- Password reset and email verification token lifecycle with simulated delivery.
- User retrieval and explicit RBAC role administration.
- Service-principal machine-token issuance on an internal route.
- PostgreSQL migrations, repositories, durable audit/outbox, Redis revocation and rate limiting.
- OpenAPI, event schema, Docker image, Make targets, Compose, Kubernetes/Helm, logs, metrics, traces, tests, and runbooks.

### 7.2 Out of Scope for MVP

- High-availability external KMS or cloud secret-manager integration; the service must expose the integration boundary and provide a secure local secret-reference mode.
- Multi-region identity replication.
- Customer self-service account profile editing beyond the documented identity fields.

## 8. Success Metrics

- **SM-1:** All FR-1 through FR-12 have automated unit, integration, security, and contract evidence before implementation completion.
- **SM-2:** Refresh-token reuse revokes the complete session family in every tested concurrent/retry scenario, validating FR-5.
- **SM-3:** No test or log inspection reveals plaintext password, refresh-token, reset-token, bearer-token, or private-key material, validating FR-2, FR-3, FR-5, and FR-12.
- **SM-4:** The service starts, passes readiness, and serves its documented health, OpenAPI, and JWKS endpoints in Docker Compose and kind, validating NFR-7 and NFR-8.
- **SM-5:** A trace links login or refresh through PostgreSQL and audit outbox publication; metrics identify authentication failures, refresh reuse, revocations, rate limiting, and outbox backlog.

**Counter-metrics:**

- **SM-C1:** Password-hash latency must not be optimized by weakening Argon2id parameters.
- **SM-C2:** Authentication success rate must not be improved by relaxing issuer, audience, algorithm, session, or authorization checks.

## 9. Open Questions

1. Which production secret manager or KMS will replace the local mounted-key provider when the platform moves beyond the portfolio environment?
2. What retention period and access policy will be selected for security audit records in a hosted deployment?
3. Which external gateway routes will expose the Identity endpoints, and which remain internal-only?

These questions do not block the local implementation because the provider, retention, and routing boundaries are explicit and configurable.

## 10. Assumptions Index

- `[ASSUMPTION: initial performance profile]` — p95 below 500 ms at 25 requests/second is a local production-like baseline, not a platform SLO.
- `[ASSUMPTION: local key provider]` — local development uses a secret-file provider behind a port so a KMS provider can replace it later.
- `[ASSUMPTION: audit retention]` — retention is configurable and will be finalized before hosted deployment.
