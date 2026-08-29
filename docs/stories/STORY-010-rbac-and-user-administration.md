# STORY-010 — Enforce RBAC and User Administration

## User / system outcome

As an authorized administrator or operator, I can manage explicit roles while customers and support users receive only permitted access.

## Context

Identity owns the role/scope catalog. Gateway checks coarse scopes, but resource ownership remains in the owning business service.

## Acceptance criteria

- [ ] Given a customer token, when a customer resource is requested, then only the self-scoped route is authorized by the consuming service.
- [ ] Given support scope, when an order read-any operation is requested, then it is allowed; a refund operation remains forbidden.
- [ ] Given finance scope, when a payment refund operation is requested, then it is allowed according to Payments policy.
- [ ] Given a caller without explicit administration permission, when a role mutation is requested, then `403` is returned.
- [ ] Given an authorized role change, when a new token is issued, then the updated explicit scopes appear and the action is audited.
- [ ] Given an administrator disables a user, when the mutation succeeds, then all active session families for that user are revoked before success is reported.

## API changes

Implement `GET /v1/users/{user_id}`, role assignment, and role removal endpoints.

## Event changes

Emit `role.assigned`, `role.removed`, `user.disabled`, and `user.enabled` audit actions.

## Data changes

Implement role, permission, role-permission, and user-role relations with uniqueness constraints.

## Failure cases

Missing scope, foreign user, unknown role, duplicate assignment, last-admin removal, and database conflict.

## Observability

Measure denied authorization by bounded reason class and audit successful/denied administrative actions safely.

## Security considerations

No magic admin bypass; protect role mutation against privilege escalation and accidental removal of the last administrative capability.

## Test plan

Authorization matrix, API, repository, contract, privilege-escalation, and audit tests.

## Out of scope

Business-service resource policies beyond their Identity-facing claims and scopes.

## Dependencies

STORY-006, STORY-007, ADR-009, ADR-010, and the role/scope catalog.

## Validation evidence

Attach the authorization matrix test report.
