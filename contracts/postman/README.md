# Identity Service Postman Tests

Import both files in this directory:

1. `identity-service.postman_collection.json`
2. `identity.local.postman_environment.json`

Start the local profile from the repository root:

```powershell
Copy-Item deploy/identity/.env.example deploy/identity/.env
docker compose --env-file deploy/identity/.env -f deploy/identity/compose.yaml up --build
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

The local test mailbox is deliberately not a production feature. It is enabled
only by `IDENTITY_TEST_MAILER_ENABLED=true`, stores tokens in process memory,
and is not registered when the flag is disabled. Production uses the SMTP
configuration documented in `deploy/identity/.env.example` and the deployment
design.
