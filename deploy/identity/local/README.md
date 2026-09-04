# Local Identity Environment

This service profile runs Identity with PostgreSQL, Redis, Kafka, and MailHog.
The complete local platform stack, including one shared OpenTelemetry Collector,
Tempo, Prometheus, Alertmanager, and Grafana for every service, is started from
`deploy/local/compose.yaml`.

```bash
# Recommended from the repository root: start both services and shared tooling.
make local-up

# Or start Identity only (creates the ignored local .env once).
make identity-compose-up

# Or start manually from this directory.
cp .env.example .env
docker compose --env-file .env -f compose.yaml up --build
```

The local `.env` file is intentionally ignored by Git. Replace the example
passwords before sharing access to the local environment.

- Identity API: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/v1/docs`
- OpenAPI contract: `http://localhost:8080/v1/openapi.yaml`
- MailHog inbox: `http://localhost:8025`
- MailHog SMTP: `localhost:1025` (container address: `mailhog:1025`)

The shared observability endpoints, credentials, dashboards, and alert rules
are documented in [`deploy/local/README.md`](../../local/README.md).

Identity currently emits structured logs to stdout, so inspect them with:

```powershell
docker compose --env-file .env -f compose.yaml logs -f identity
```

The OpenTelemetry Collector forwards received traces to Tempo and keeps a
debug exporter for local inspection. Use Grafana Explore with the Tempo
datasource when trace instrumentation emits spans.

Use the Postman collection in `contracts/postman`. The collection reads the
verification and password-reset tokens from the MailHog API, then calls the
Identity confirmation endpoints.
