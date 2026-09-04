# Local application deployment

This Compose file contains the local application and application-dependency
layer for StreamWeave: Identity, Gateway, PostgreSQL, Redis, Kafka, and
MailHog. Shared observability is defined separately in
`monitoring/local/compose.yaml`.

Run from the repository root:

```bash
copy monitoring/local/.env.example monitoring/local/.env
make local-up
```

The root Makefile merges this file with the monitoring Compose file into one
local project. The copied `.env` is local-only and ignored by Git. Replace the
development Gateway service token when testing authenticated downstream routes.

| Component | URL | Credentials |
|---|---|---|
| Identity API | `http://localhost:8080` | API contract |
| Gateway API | `http://localhost:8081` | API contract |
| Gateway Swagger UI | `http://localhost:8081/v1/docs` | none |
| MailHog | `http://localhost:8025` | none |

Use `make local-config` to validate the merged Compose model, `make local-logs`
to follow all application and monitoring logs, and `make local-down` to stop
the complete stack while preserving named volumes. See
`monitoring/local/README.md` for monitoring URLs and credentials.
