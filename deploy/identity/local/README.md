# Local Identity Environment

This profile runs Identity with PostgreSQL, Redis, Kafka, OpenTelemetry, Tempo,
Prometheus, Alertmanager, Grafana, and MailHog. MailHog is a real local SMTP
server: Identity sends the same multipart HTML/text emails used by the SMTP
deployment rather than logging or simulating tokens.

```powershell
Copy-Item .env.example .env
docker compose --env-file .env -f compose.yaml up --build
```

- Identity API: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/v1/docs`
- OpenAPI contract: `http://localhost:8080/v1/openapi.yaml`
- MailHog inbox: `http://localhost:8025`
- MailHog SMTP: `localhost:1025` (container address: `identity-mailhog:1025`)
- Grafana dashboard: `http://localhost:3001`
- Prometheus: `http://localhost:9090`
- Alertmanager: `http://localhost:9093`
- Tempo API/health: `http://localhost:3200/ready` (Tempo has no standalone web UI; query traces from Grafana Explore)

Grafana logs in with `IDENTITY_GRAFANA_ADMIN_USER` and
`IDENTITY_GRAFANA_ADMIN_PASSWORD` from `.env`. The provisioned dashboard is
`Identity / Identity Service Overview`. Prometheus automatically scrapes the
Identity `/metrics` endpoint and evaluates the rules in
`prometheus/rules.yml`; alerts are visible in Alertmanager and Grafana.

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
