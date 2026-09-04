# Shared local platform

This is the canonical local Compose profile for StreamWeave. It starts both
implemented services and one shared platform layer:

- Identity, Gateway, PostgreSQL, Redis, Kafka, and MailHog;
- one OpenTelemetry Collector and Tempo instance;
- one Prometheus and Alertmanager instance scraping both services;
- one Grafana instance with Identity and Gateway dashboards.

Run from the repository root:

```bash
copy deploy/local/.env.example deploy/local/.env
make local-up
```

The copied `.env` is local-only and ignored by Git. Replace the development
Gateway service token when testing authenticated downstream routes.

| Component | URL | Credentials |
|---|---|---|
| Identity API | `http://localhost:8080` | API contract |
| Gateway API | `http://localhost:8081` | API contract |
| Gateway Swagger UI | `http://localhost:8081/v1/docs` | none |
| Grafana | `http://localhost:3000` | `IDENTITY_GRAFANA_ADMIN_USER` / `IDENTITY_GRAFANA_ADMIN_PASSWORD` |
| Prometheus | `http://localhost:9090` | none |
| Alertmanager | `http://localhost:9093` | none |
| Tempo readiness | `http://localhost:3200/ready` | none |
| MailHog | `http://localhost:8025` | none |

Use `make local-config` to validate the rendered Compose model, `make local-logs`
to follow all service logs, and `make local-down` to stop the stack while
preserving named volumes.
