# Identity Service Deployment and Operations Design

## Deployment Contract

Identity must behave the same across local Docker Compose and kind/Helm. Configuration is injected through environment variables and secret references; images and ConfigMaps contain no credentials or private keys.

Required configuration names:

| Variable | Purpose | Secret |
|---|---|---|
| `IDENTITY_HTTP_ADDR` | Listen address. | No |
| `IDENTITY_ISSUER` | JWT issuer claim and verification policy. | No |
| `IDENTITY_HUMAN_AUDIENCE` | Human access-token audience. | No |
| `IDENTITY_MACHINE_AUDIENCE` | Service-token audience. | No |
| `IDENTITY_DATABASE_URL` | Identity PostgreSQL connection. | Yes |
| `IDENTITY_REDIS_URL` | Disposable revocation/rate-limit Redis connection. | Yes if authenticated |
| `IDENTITY_KAFKA_BROKERS` | Audit event broker addresses. | No |
| `IDENTITY_SIGNING_KEY_PATH` | Mounted active private-key reference. | Path only; file is secret |
| `IDENTITY_SIGNING_KEY_ID` | Active public key ID. | No |
| `IDENTITY_DEMO_REGISTRATION_ENABLED` | Demo-registration feature flag; false in production. | No |
| `IDENTITY_OTEL_ENDPOINT` | OTLP collector endpoint. | No |
| `IDENTITY_LOG_LEVEL` | Structured log level. | No |
| `IDENTITY_MAILER_MODE` | `smtp` for local MailHog, staging Mailtrap Sandbox, and production Mailtrap Email Sending. | No |
| `IDENTITY_TEST_MAILER_ENABLED` | Enables the in-memory mailbox only for isolated local tests; false for the MailHog profile and all shared environments. | No |
| `IDENTITY_SMTP_HOST` / `IDENTITY_SMTP_PORT` | SMTP relay endpoint. | No |
| `IDENTITY_SMTP_USERNAME` / `IDENTITY_SMTP_PASSWORD` | SMTP credentials. | Username no; password yes |
| `IDENTITY_SMTP_FROM` | Verified sender address. | No |
| `IDENTITY_PUBLIC_APP_URL` | Base URL used in verification and reset links. | No |
| `IDENTITY_EMERGENCY_REVOCATION_ENABLED` | Enable fail-closed JTI denylist enforcement. | No |
| `IDENTITY_RATE_LOGIN_FAILURES` | Login failures allowed per 15-minute window; default `5`. | No |
| `IDENTITY_RATE_REFRESH_PER_MINUTE` | Refresh attempts per session family per minute; default `30`. | No |
| `IDENTITY_RATE_RESET_PER_HOUR` | Password-reset requests per account and source prefix per hour; default `3`. | No |
| `IDENTITY_RATE_SERVICE_TOKEN_PER_MINUTE` | Service-token requests per principal and source prefix per minute; default `30`. | No |

Initial rate-limit defaults are configuration, not hard-coded policy: five failed login attempts per 15 minutes per account and source prefix, 30 refresh attempts per minute per session family, three password-reset requests per hour per account and source prefix, and 30 service-token requests per minute per principal and source prefix. Responses use `429` after a limit is exceeded. Redis outage follows the fail-closed credential-issuance policy in the PRD and architecture design.

## Make and Script Contract

The implementation phase must make these commands canonical rather than duplicating them in documentation:

```text
make identity-generate
make identity-lint
make identity-test
make identity-test-integration
make identity-test-security
make identity-build
make identity-image
make identity-compose-up
make identity-compose-down
make identity-kind-deploy
make identity-smoke
```

The commands are documentation targets until their Makefile, scripts, and workflows exist. CI must run the same targets.

## Docker Compose Profile

The Identity profile starts PostgreSQL, Redis, Kafka, the OpenTelemetry Collector, and Identity. PostgreSQL uses a dedicated database/user; no other service receives the Identity connection string. The profile includes health checks and waits for dependency readiness without treating Redis as durable state.

Local Compose uses MailHog as a real SMTP server so verification and reset emails can be inspected in the MailHog UI. An in-memory simulated mailbox remains available only for isolated tests when `IDENTITY_TEST_MAILER_ENABLED=true`; that mode is never enabled in shared environments or production. Staging uses Mailtrap Email Sandbox, and production uses Mailtrap Email Sending with `IDENTITY_SMTP_HOST`, `IDENTITY_SMTP_PORT`, `IDENTITY_SMTP_USERNAME`, `IDENTITY_SMTP_PASSWORD`, and `IDENTITY_SMTP_FROM`; credentials remain Secret-backed. Compose logs are structured JSON and can be inspected without exposing credentials. The Identity container runs as a non-root user, uses a multi-stage build, and receives signing material through a local secret reference excluded from Git.

## Kubernetes and Helm Profile

The Helm deployment must provide:

- a dedicated ServiceAccount and least-privilege NetworkPolicy;
- ClusterIP service with no public service-token route;
- Secret references for database, Redis, and signing-key provider configuration;
- ConfigMap for non-secret issuer, audience, feature flags, and telemetry settings;
- startup, liveness, and readiness probes;
- non-root security context and read-only filesystem where compatible;
- CPU/memory requests and limits;
- PodDisruptionBudget and a local one-replica-safe profile;
- migration job or controlled migration hook with expand/migrate/contract compatibility;
- graceful termination period long enough to finish current HTTP work and flush telemetry;
- explicit dependency egress only to PostgreSQL, Redis, Kafka, OTLP, and the configured SMTP relay.

Readiness must fail when Identity cannot perform its responsibility, including missing signing configuration or PostgreSQL. Redis degradation is surfaced separately and follows the fail-closed revocation policy.

Operational endpoints are `/health/live` for process liveness, `/health/ready` for dependency readiness, `/health/startup` for initialization completion, and `/metrics` for Prometheus scraping. They are unauthenticated, network-restricted at deployment boundaries where appropriate, and must not expose configuration values or sensitive dependency details.

## Logging and Monitoring

### Logs

Structured JSON events include timestamp, level, service, message, request ID, trace ID, span ID, correlation ID, route, outcome, actor ID when authenticated, and bounded reason code. Passwords, bearer tokens, refresh tokens, reset tokens, email delivery content, private keys, and full IP addresses are prohibited.

### Metrics

The initial dashboard covers authentication outcomes, refresh reuse, session revocation, JWT verification, rate limiting, outbox backlog/publication, database pool pressure, readiness, and dependency latency. Labels remain bounded and never contain email, user ID, token ID, or arbitrary error text.

### Traces

HTTP spans connect to password verification, PostgreSQL transaction, session transition, audit/outbox insert, Kafka publication, and SMTP mail delivery. Sensitive request fields are excluded from span attributes.

### Alerts

Create alerts for sustained readiness failure, authentication failure spikes, refresh-token reuse, outbox backlog growth, Kafka publication failure, database pool exhaustion, key-provider failure, and unexpected service-token issuance volume.

## Release and Recovery

Identity images are built from reviewed `main` commits, tagged with the platform version, and published with SBOM/provenance when the platform release gate is operational. Migrations are backward-compatible. Rollback uses an older compatible image; database rollback is not assumed.

Operational drills must demonstrate key rotation, database recovery, Redis outage, outbox recovery, pod restart, and secret rotation using the runbooks in `docs/runbooks/`.
