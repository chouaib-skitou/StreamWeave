# Orders Deployment Design

## Local

Orders will join the canonical `deploy/local/compose.yaml` stack with host port `8082`, container port `8080`, its own `orders_db` logical database/user, and shared Kafka/Redis/telemetry. MailHog is not an Orders dependency. No duplicate monitoring Compose tree is permitted.

## Staging and production

The `deploy/orders/` chart uses a ClusterIP Service, non-root Deployment(s), external Secret references, TLS PostgreSQL/Kafka, a PostgreSQL-backed API readiness probe, a process-local relay probe, PDB, and resource requests/limits. The API and relay are separate workloads with separate service accounts and Kafka credentials: the API has no Kafka access, while the relay has only the topic ACLs listed in `orders-kafka.md`.

NetworkPolicy allows only Gateway ingress, approved monitoring scrape, Orders PostgreSQL, Kafka, and telemetry. When JWKS endpoints are external, the environment overlay explicitly permits TLS egress on port 443; network policy cannot express DNS hostnames, so the cluster egress boundary must additionally be enforced by the platform firewall.

## Lifecycle

Migrations run as a controlled pre-deploy job with a PostgreSQL advisory lock and expand/migrate/contract compatibility. Shutdown stops new HTTP mutations, drains Kafka handlers within a deadline, leaves uncommitted work for retry, and never acknowledges a message before its transaction commits.

## Artifact and release

The release workflow publishes `ghcr.io/chouaib-skitou/streamweave-orders:v<version>` with immutable version/SHA tags, SBOM, and provenance. Staging and production Helm overlays require an image digest at deployment time; mutable tags are rejected by the chart. Secrets and kubeconfig never enter Git.
