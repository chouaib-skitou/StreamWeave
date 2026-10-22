# StreamWeave Postman Tests

## Unified collection

The recommended entry point is the shared collection and environment:

1. `streamweave.postman_collection.json`
2. `streamweave.local.postman_environment.json`

The collection is organized as `Variables`, `Identity Service`, `Gateway
Service`, `Orders Service`, `Inventory Service`, `Payments Service`,
`Fulfillment Service`, and `Notifications Service`. The last four folders are
reserved for their service contracts and are intentionally empty until those
services are implemented. Runtime values and tokens are stored in the active
Postman environment; real credentials must never be committed.

Run the Identity flow first, then the Gateway order flow, and finally the
Orders contract checks. The collection extracts access and refresh tokens from
the login response and stores the created order identifier for subsequent
requests.

The service-specific collections below remain available for exhaustive route
coverage and advanced RBAC, password-reset, and internal-contract scenarios.

## Identity Service

Import both files in this directory:

1. `identity-service.postman_collection.json`
2. `identity.local.postman_environment.json`

Start the canonical shared local profile from the repository root:

```powershell
Copy-Item monitoring/local/.env.example monitoring/local/.env
docker compose --env-file monitoring/local/.env -f deploy/local/compose.yaml -f monitoring/local/compose.yaml up -d --build
```

Run the requests in this order for the complete human-user flow:

1. Operations health checks
2. Register demo user
3. Read latest verification mail (local only)
4. Confirm email verification
5. Login
6. Refresh tokens
7. List sessions
8. Request password reset
9. Read latest password reset mail (local only)
10. Confirm password reset
11. Login after password reset
12. Logout and logout all sessions

The login and refresh test scripts automatically update `access_token`,
`refresh_token`, and `session_id`. The registration and mailbox scripts update
`user_id`, `verification_token`, and `reset_token`.

The user/RBAC requests require an access token containing
`identity:roles:manage` for successful role and status mutations. The service
token request requires an active service principal and credential seeded in the
Identity database; its variables are placeholders by design.

Local uses MailHog as a real SMTP server. Open `http://localhost:8025` to
inspect the rendered HTML and text versions. Staging and production use the
SMTP configurations documented in `deploy/identity/staging` and
`deploy/identity/production`.

## Orders Service

Import both files:

1. `orders-service.postman_collection.json`
2. `orders.local.postman_environment.json`

The collection covers the three local backend health checks, Gateway setup and
authentication, every Orders operation route, the public Gateway order flow,
and the internal Orders contract. Set `gateway_service_token` to the same
local value as `GATEWAY_SERVICE_TOKEN` in `monitoring/local/.env`. Run the
`Local setup` folder first. It creates a development customer, extracts the
real verification token from MailHog, verifies the email, and stores the access
and refresh tokens for subsequent requests. The demo account is local-only and
is not a production seed.
