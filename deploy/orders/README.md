# Orders Deployment Contract

This directory is reserved for the Orders Dockerfile and Helm chart to be added during implementation. The approved deployment behavior is documented in [`../../docs/design/orders-deployment.md`](../../docs/design/orders-deployment.md).

Orders runs on container port `8080`; local host mapping is `8082` because Identity uses `8080` and Gateway uses `8081`. Staging and production use an internal ClusterIP and TLS-protected dependencies. Monitoring remains centralized under [`../../monitoring/`](../../monitoring/).

The implementation must add the chart, migration job, ServiceAccount, external Secret references, probes, PDB, resource limits, HPA/KEDA policy, and restrictive NetworkPolicy without committing credentials or environment files.
