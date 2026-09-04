# Local shared monitoring

This directory contains the single local observability stack shared by
Identity, Gateway, and future StreamWeave services.

The application services are defined in `deploy/local/compose.yaml`; the
monitoring services are defined here. The root Makefile starts both files as
one Compose project:

```bash
copy monitoring/local/.env.example monitoring/local/.env
make local-up
```

The copied `.env` is ignored and must never be committed. Use `make
local-config` to validate the merged model, `make local-logs` to follow logs,
and `make local-down` to stop the stack without deleting volumes.

| Component | URL |
|---|---|
| Identity API | `http://localhost:8080` |
| Identity Swagger UI | `http://localhost:8080/v1/docs` |
| Gateway API | `http://localhost:8081` |
| Gateway Swagger UI | `http://localhost:8081/v1/docs` |
| Grafana | `http://localhost:3000` |
| Prometheus | `http://localhost:9090` |
| Alertmanager | `http://localhost:9093` |
| Tempo readiness | `http://localhost:3200/ready` |
| MailHog | `http://localhost:8025` |
