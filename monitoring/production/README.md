# Production monitoring

Production uses one shared monitoring platform for the entire StreamWeave
environment, isolated in a dedicated monitoring namespace. Application charts
must never install their own monitoring stack.

The production release is required to provide:

- highly available Prometheus and Alertmanager with durable storage;
- durable, access-controlled trace storage for Tempo or its production backend;
- external secrets, TLS, SSO/RBAC, and network policies;
- explicit retention, resource requests/limits, backup, and restore procedures;
- alert routing with deduplication, inhibition, escalation, and a tested
  on-call destination;
- dashboards and alert rules covering all deployed services.

Production promotion is blocked until these environment values and operational
runbooks are supplied by the target Kubernetes platform.
