# Orders Authentication and Authorization Context

## Trust chain

```text
Human JWT → Gateway verifies Identity token
Gateway service token → Orders authenticates Gateway principal
Trusted actor context → Orders enforces resource/business authorization
```

Orders accepts public requests only through Gateway. It validates the Gateway token's issuer, audience, subject, expiry, algorithm, machine principal, and required service scope. Only then may it read the bounded actor context headers supplied by Gateway.

## Safe context

Allowed context is actor subject (`usr_...`), actor type, normalized scopes, active session reference if the contract supports it, request ID, correlation ID, and W3C trace context. Human bearer tokens, passwords, cookies, emails, and arbitrary client headers are never forwarded or logged.

## Permission model

Canonical Orders permissions:

- `orders:read:self`
- `orders:read:any`
- `orders:write:self`
- `orders:write:any`
- `orders:cancel:self`
- `orders:cancel:any`

The existing Gateway/Identity baseline uses `orders:write:self` for creation and `orders:write:any` for operator creation. This v1 contract keeps that vocabulary; the blueprint's earlier `orders:create` example is retired to avoid two names for one permission.

`admin` is not a magical bypass; Identity maps roles to explicit permissions. Gateway performs the coarse route check, while Orders verifies ownership and state. The `customer_id` request field must equal the trusted actor for self permissions. Operator permissions are explicit and auditable.

## Enumeration policy

For a customer request, an order not in the actor's visible set returns the same safe `404` shape as an unknown order. Operator denial is `403` only when the actor is authenticated but lacks the operation permission. No response leaks another customer's existence.

## Security tests required

Test forged context headers, invalid Gateway token, missing scope, mismatched customer ID, cross-customer get/list/cancel, support read-any, finance-only restrictions, oversized headers, stale session context, and machine-token expiry.
