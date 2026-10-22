# Orders Deployment Contract

This directory contains the Orders production Docker image and Helm chart. The approved deployment behavior is documented in [`../../docs/design/orders-deployment.md`](../../docs/design/orders-deployment.md).

Orders runs on container port `8080`; local host mapping is `8082` because Identity uses `8080` and Gateway uses `8081`. Staging and production use an internal ClusterIP and TLS-protected dependencies. Monitoring remains centralized under [`../../monitoring/`](../../monitoring/).

The chart includes the migration hook, separate API and relay Deployments, ServiceAccount, external Secret references, probes, PDB, resource limits, and restrictive NetworkPolicy. Credentials and environment files are never committed. Horizontal API scaling is supported; the relay remains a singleton consumer group member unless partitioned relay deployment is explicitly enabled by the platform topology.
