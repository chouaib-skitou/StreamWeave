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

The image must be minimal, deterministic, non-root, read-only-root-filesystem compatible, and contain no credentials. Gateway listens on container port `8080`, matching the service convention without requiring a host-port collision. Health probes use `/health/live` and `/health/ready`; metrics are bound to an internal interface or protected network surface.

Local Compose maps `localhost:8081` to Gateway container port `8080`, because Identity already maps `localhost:8080` to its own container port `8080`. Kubernetes uses separate ClusterIP Services, so Gateway and Identity may both target container port `8080` without a conflict.

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

## CI and release contract

Gateway changes must activate a dedicated Gateway verification job through path-aware CI. The job runs formatting, static analysis, unit tests with race detection, the minimum 85% application coverage gate, OpenAPI validation, security tests, a container build, Helm lint, and local Compose configuration validation. It must not duplicate the complete Identity job when Identity files are unchanged.

The main-branch release workflow remains the single publisher. When a release is created, it publishes the OCI image as `ghcr.io/chouaib-skitou/streamweave-gateway:v<version>` with the platform version tag, immutable commit tag, SBOM, and provenance. Gateway release metadata follows the existing `VERSION`/`CHANGELOG.md` synchronization and never stores registry credentials in the repository.

## Local development

Local Compose should run Gateway with Identity, Orders dependencies required by the current milestone, Redis, and the observability stack. Mail delivery belongs to Identity/MailHog, not Gateway. Local configuration must be reproducible from `.env.example` without committing a real `.env`.
