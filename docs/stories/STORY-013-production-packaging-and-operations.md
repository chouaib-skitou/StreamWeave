# STORY-013 — Package and Operate Identity in Production-Like Environments

## User / system outcome

As an operator, I can run, observe, secure, upgrade, and recover Identity in Docker Compose and Kubernetes without exposing secrets or losing durable sessions.

## Context

The service is not complete until packaging, migrations, probes, logging, metrics, tracing, resource controls, and recovery procedures are executable and evidenced.

## Acceptance criteria

- [ ] Given Docker Compose, when the platform starts, then PostgreSQL, Redis, telemetry, and Identity become healthy with documented configuration.
- [ ] Given the Helm chart, when it is linted and deployed to kind, then probes, secret references, service account, NetworkPolicy, PDB, resources, and non-root security context are valid.
- [ ] Given a request and authentication failure, when logs and traces are inspected, then correlation is present and secrets are absent.
- [ ] Given PostgreSQL, Redis, Kafka, or key-provider outage, when the failure runbook is followed, then the expected fail-closed/degraded behavior is reproducible.
- [ ] Given a compatible migration, when a new version is rolled forward, then active sessions and old compatible readers remain safe.
- [ ] Given the release gate, when tests, contracts, linting, security scanning, and Helm validation pass, then artifacts are eligible for release.

## API changes

Document operational endpoints, configuration, and deployment exposure.

## Event changes

Document relay metrics, DLQ/retry behavior, and security-audit delivery operations.

## Data changes

Add migration, backup/restore, cleanup, and retention procedures.

## Failure cases

Dependency outage, pod restart, readiness failure, outbox backlog, duplicate event, key rotation, migration failure, and rollback.

## Observability

Provide low-cardinality metrics, structured logs, OTLP traces, dashboards, and alerts for auth failures, refresh reuse, revocation, outbox, DB pool, and readiness.

## Security considerations

Non-root container, secret references, no public machine-token route, restricted network paths, immutable release artifacts, and dependency/image scanning.

## Test plan

Compose smoke, kind deployment, health, log redaction, trace propagation, failure injection, migration, and security scans.

## Out of scope

Cloud provider-specific KMS, multi-region identity, and external email infrastructure.

## Dependencies

STORY-006 through STORY-012, the deployment design, and all Identity runbooks.

## Validation evidence

Attach Compose and kind output, scan reports, dashboards, runbook drills, and rollback evidence.
