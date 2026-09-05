# Orders Deployment Design

## Local

Orders will join the canonical `deploy/local/compose.yaml` stack with host port `8082`, container port `8080`, its own `orders_db` logical database/user, and shared Kafka/Redis/telemetry. MailHog is not an Orders dependency. No duplicate monitoring Compose tree is permitted.

## Staging and production

The `deploy/orders/` chart will use a ClusterIP Service, non-root Deployment(s), external Secret references, TLS PostgreSQL/Kafka, readiness that reflects required local dependencies, liveness independent of dependencies, startup protection, PDB, resource requests/limits, and HPA/KEDA according to API/consumer load. The API and relay are separate workloads with separate service accounts and Kafka credentials: the API has no Kafka access, while the relay has only the topic ACLs listed in `orders-kafka.md`.

NetworkPolicy allows only Gateway ingress, approved monitoring scrape, Orders PostgreSQL, Kafka, Redis when configured, Inventory/Payments/Fulfillment endpoints, and telemetry. There is no public Orders ingress or arbitrary internet egress.

## Lifecycle

Migrations run as a controlled pre-deploy job with advisory locking and expand/migrate/contract compatibility. Shutdown stops new HTTP mutations, drains Kafka handlers within a deadline, leaves uncommitted work for retry, and never acknowledges a message before its transaction commits.

## Artifact and release

The release workflow must be extended when Orders is implemented to publish `ghcr.io/chouaib-skitou/streamweave-orders:v<version>` with immutable version/SHA tags, SBOM, provenance, and vulnerability checks. Deployment pins a digest, not a mutable `main` tag. Secrets and kubeconfig never enter Git.
