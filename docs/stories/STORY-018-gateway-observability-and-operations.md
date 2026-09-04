# STORY-018 — Operate Gateway with production-grade telemetry and deployment

## User / system outcome

Operators can deploy, observe, alert, scale, and recover Gateway without exposing secrets or relying on undocumented behavior.

## Context

Gateway is a stateless public edge whose outages can affect every client. Operational contracts must exist before code is written.

## Acceptance criteria

- [x] Given a local environment, when the documented Compose command runs, then health, metrics, logs, and traces are inspectable without committed secrets.
- [x] Given a Kubernetes deployment, when manifests are rendered, then non-root security, probes, resources, PDB, HPA, and restrictive network policy are present.
- [x] Given a Gateway change, when path-aware CI runs, then Gateway quality checks execute without unnecessarily duplicating unrelated service suites.
- [x] Given a release on `main`, when the release workflow publishes images, then `streamweave-gateway:v<version>` is built with SBOM/provenance and the existing registry permissions.
- [x] Given telemetry, when logs/metrics/traces are inspected, then no token, password, cookie, body, or unbounded identifier is present.
- [x] Given sustained errors/latency/JWKS/limiter failures, when alert rules evaluate, then an operator can identify the route class and dependency.
- [x] Given an outage, when the relevant runbook is followed, then recovery does not disable authentication or rate-limit safety controls.

## API changes

Health, metrics, correlation, and Problem Details contracts are documented.

## Event changes

None.

## Data changes

None.

## Failure cases

Readiness loss, upstream outage, Redis outage, JWKS outage, overload, rollout failure, telemetry exporter failure.

## Observability

Metrics/logs/traces and dashboard/alert vocabulary are defined in `gateway-observability.md`.

## Security considerations

Internal-only metrics, redaction, least-privilege network policy, no secret files, non-root image.

## Test plan

Container smoke tests, Helm lint/render, health probe tests, network-policy review, telemetry redaction tests, and failure-injection checks.

## Dependencies

Container/Helm conventions, OTel/Prometheus/Grafana/Loki/Tempo stack, CI release policy, and Gateway runbooks.

## Out of scope

Cloud-provider-specific managed ingress implementation and multi-region traffic management.

## Validation evidence

Documentation gate: deployment, observability, security checklist, and four Gateway runbooks reviewed as one operational contract.
