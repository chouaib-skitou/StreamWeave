# StreamWeave monitoring architecture

StreamWeave uses one shared observability platform per environment. Services
publish telemetry and expose metrics; they do not own a private copy of
Prometheus, Alertmanager, Grafana, Tempo, or the OpenTelemetry Collector.

## Environment layout

- `monitoring/local/`: one Docker Compose observability stack for local development.
- `monitoring/staging/`: the shared staging monitoring release contract.
- `monitoring/production/`: the shared production monitoring release contract.
- `deploy/<service>/`: application images and application Kubernetes charts.
- `deploy/local/`: the single local application platform Compose definition.

This separation keeps service deployment concerns independent from platform
observability while retaining a single monitoring boundary for all services in
an environment.

## Data flow

```text
services -> OpenTelemetry Collector -> Tempo
services -> /metrics -> Prometheus -> Alertmanager
Prometheus, Alertmanager, Tempo -> Grafana
```

Local uses one Compose project so service DNS names resolve across both files.
Staging and production must use one monitoring namespace/release per cluster or
environment, with external secrets, durable storage, TLS, access control,
retention, and alert routing configured for that environment.
