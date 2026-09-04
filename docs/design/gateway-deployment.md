# Gateway deployment design

## Environment contract

Environment-specific files belong under `deploy/gateway/local`, `deploy/gateway/staging`, and `deploy/gateway/production`. Commit only examples and non-sensitive defaults. Secrets are injected by the environment's secret manager or Kubernetes Secret reference; `.env` files and kubeconfigs remain ignored.

Required configuration categories:

- public listen address and request/body limits;
- Identity issuer, expected human audience, JWKS URL, allowed algorithm, and clock skew;
- Gateway machine audience, service credential reference, and upstream Identity/Orders URLs;
- Redis address, TLS policy, key prefix, and route limit policy;
- per-route timeout and retry policy;
- CORS allowlist (empty by default);
- OTLP endpoint, metrics binding, log level, and environment name.

## Container

The image must be minimal, deterministic, non-root, read-only-root-filesystem compatible, and contain no credentials. It exposes only the Gateway HTTP port. Health probes use `/health/live` and `/health/ready`; metrics are bound to an internal interface or protected network surface.

## Kubernetes

The production deployment must provide:

- Deployment, ClusterIP Service, ServiceAccount, ConfigMap, Secret references, probes, resource requests/limits, PDB, HPA, and restrictive NetworkPolicy;
- ingress only from the approved edge/load-balancer path;
- egress only to Identity, Orders, Redis, and the configured telemetry collector;
- no PostgreSQL, Kafka, or arbitrary internet egress;
- security context with non-root, dropped capabilities, seccomp default, and no privilege escalation;
- separate staging and production values, with TLS enforced for Redis and all internal HTTP where supported.

## Scaling and rollout

Gateway instances are stateless except for disposable in-memory caches. Scale horizontally using CPU plus request/in-flight signals once those metrics are available. Readiness must drain new traffic before termination. Rolling updates must preserve at least one healthy instance and respect the PDB.

## Local development

Local Compose should run Gateway with Identity, Orders dependencies required by the current milestone, Redis, and the observability stack. Mail delivery belongs to Identity/MailHog, not Gateway. Local configuration must be reproducible from `.env.example` without committing a real `.env`.
