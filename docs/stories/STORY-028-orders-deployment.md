# STORY-028 — Package and deploy Orders

## Outcome

Run Orders safely in local Compose, staging, and production Kubernetes environments.

## Acceptance criteria

- [ ] Container is non-root, minimal, pinned, read-only, and emits SBOM/provenance.
- [ ] Local Orders uses port `8082` and shared monitoring dependencies.
- [ ] Helm includes probes, resources, PDB, ServiceAccount, external Secrets, and restrictive NetworkPolicy.
- [ ] Production PostgreSQL/Kafka/telemetry connections require TLS.
- [ ] Deployment uses immutable image digest and compatible migrations.

## Test plan

Docker build, Compose config, Helm lint/template, kind smoke, and policy validation.

## API and event changes

Expose Orders on the reserved internal port `8082`, with separate API/relay health semantics and no public ingress. Deployment configuration must preserve the versioned REST/Kafka contracts.

## Data and failure behavior

Run compatible migrations before serving traffic and preserve PostgreSQL, Kafka, and Outbox recovery guarantees across rollout. Readiness fails when required dependencies are unavailable; liveness remains process-local.

## Observability and security

Use the shared monitoring stack and central alert rules. Images run non-root with read-only filesystems, pinned dependencies, TLS-required production endpoints, external secrets, restricted egress, and immutable digests.

## Dependencies and out of scope

Depends on platform Compose/Helm conventions, release workflow, and Orders health contract. Cloud-provider-specific ingress, KMS, and multi-region topology are out of scope.
